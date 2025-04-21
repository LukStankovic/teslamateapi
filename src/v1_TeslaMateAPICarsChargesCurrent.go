package main

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// TeslaMateAPICarsChargesCurrentV1 func
func TeslaMateAPICarsChargesCurrentV1(c *gin.Context) {

	// define error messages
	var (
		CarsChargesCurrentError1 = "Unable to load current charge."
		CarsChargesCurrentError2 = "Unable to load current charge details."
	)

	// getting CarID param from URL
	CarID := convertStringToInteger(c.Param("CarID"))

	// creating structs for /cars/<CarID>/charges/current
	// Car struct - child of Data
	type Car struct {
		CarID   int        `json:"car_id"`   // smallint
		CarName NullString `json:"car_name"` // text (nullable)
	}
	// BatteryDetails struct - child of Charge
	type BatteryDetails struct {
		StartBatteryLevel   int `json:"start_battery_level"`   // int
		CurrentBatteryLevel int `json:"current_battery_level"` // int
	}
	// PreferredRange struct - child of Charge
	type PreferredRange struct {
		StartRange   float64 `json:"start_range"`   // float64
		CurrentRange float64 `json:"current_range"` // float64
	}
	// ChargerDetails struct - child of ChargeDetails
	type ChargerDetails struct {
		ChargerActualCurrent int `json:"charger_actual_current"` // int
		ChargerPhases        int `json:"charger_phases"`         // int
		ChargerPilotCurrent  int `json:"charger_pilot_current"`  // int
		ChargerPower         int `json:"charger_power"`          // int
		ChargerVoltage       int `json:"charger_voltage"`        // int
	}
	// FastChargerInfo struct - child of ChargeDetails
	type FastChargerInfo struct {
		FastChargerPresent bool           `json:"fast_charger_present"` // bool
		FastChargerBrand   sql.NullString `json:"fast_charger_brand"`   // string
		FastChargerType    string         `json:"fast_charger_type"`    // string
	}
	// BatteryInfo struct - child of ChargeDetails
	type BatteryInfo struct {
		IdealBatteryRange    float64  `json:"ideal_battery_range"`     // float64
		RatedBatteryRange    float64  `json:"rated_battery_range"`     // float64
		BatteryHeater        bool     `json:"battery_heater"`          // bool
		BatteryHeaterOn      bool     `json:"battery_heater_on"`       // bool
		BatteryHeaterNoPower NullBool `json:"battery_heater_no_power"` // bool
	}
	// ChargeDetails struct - child of Charge
	type ChargeDetails struct {
		DetailID             int             `json:"detail_id"`                // integer
		Date                 string          `json:"date"`                     // string
		BatteryLevel         int             `json:"battery_level"`            // int
		UsableBatteryLevel   int             `json:"usable_battery_level"`     // int
		ChargeEnergyAdded    float64         `json:"charge_energy_added"`      // float64
		NotEnoughPowerToHeat NullBool        `json:"not_enough_power_to_heat"` // bool
		ChargerDetails       ChargerDetails  `json:"charger_details"`          // struct
		BatteryInfo          BatteryInfo     `json:"battery_info"`             // struct
		ConnChargeCable      string          `json:"conn_charge_cable"`        // string
		FastChargerInfo      FastChargerInfo `json:"fast_charger_info"`        // struct
		OutsideTemp          float64         `json:"outside_temp"`             // float64
	}
	// Charge struct - child of Data
	type Charge struct {
		ChargeID          int             `json:"charge_id"`           // int
		StartDate         string          `json:"start_date"`          // string
		EndDate           string          `json:"end_date"`            // string
		IsCharging        bool            `json:"is_charging"`         // bool
		Address           string          `json:"address"`             // string
		ChargeEnergyAdded float64         `json:"charge_energy_added"` // float64
		ChargeEnergyUsed  float64         `json:"charge_energy_used"`  // float64
		Cost              float64         `json:"cost"`                // float64
		DurationMin       int             `json:"duration_min"`        // int
		DurationStr       string          `json:"duration_str"`        // string
		BatteryDetails    BatteryDetails  `json:"battery_details"`     // BatteryDetails
		RangeIdeal        PreferredRange  `json:"range_ideal"`         // PreferredRange
		RangeRated        PreferredRange  `json:"range_rated"`         // PreferredRange
		OutsideTempAvg    float64         `json:"outside_temp_avg"`    // float64
		Odometer          float64         `json:"odometer"`            // float64
		ChargeDetails     []ChargeDetails `json:"charge_details"`      // struct
	}
	// TeslaMateUnits struct - child of Data
	type TeslaMateUnits struct {
		UnitsLength      string `json:"unit_of_length"`      // string
		UnitsTemperature string `json:"unit_of_temperature"` // string
	}
	// Data struct - child of JSONData
	type Data struct {
		Car            Car            `json:"car"`
		Charge         Charge         `json:"charge"`
		TeslaMateUnits TeslaMateUnits `json:"units"`
	}
	// JSONData struct - main
	type JSONData struct {
		Data Data `json:"data"`
	}

	// creating required vars
	var (
		CarName                       NullString
		charge                        Charge
		ChargeDetailsData             []ChargeDetails
		UnitsLength, UnitsTemperature string
		isCharging                    bool
	)

	// Create temp vars to handle NULL values in the database
	var (
		startIdealRange, currentIdealRange        sql.NullFloat64
		startRatedRange, currentRatedRange        sql.NullFloat64
		startBatteryLevel, currentBatteryLevel    sql.NullInt64
		chargeEnergyAdded, chargeEnergyUsed, cost sql.NullFloat64
		outsideTempAvg                            sql.NullFloat64
		odometer                                  sql.NullFloat64
		durationMin                               sql.NullFloat64 // Changed from sql.NullInt64 to sql.NullFloat64
		durationStr, address                      sql.NullString
		endDate                                   sql.NullString
	)

	// Get the most recent charging process for this car, prioritizing charges in progress
	query := `
		SELECT
			charging_processes.id AS charge_id,
			start_date,
			end_date,
			COALESCE(geofence.name, CONCAT_WS(', ', COALESCE(address.name, nullif(CONCAT_WS(' ', address.road, address.house_number), '')), address.city)) AS address,
			COALESCE(charging_processes.charge_energy_added, 0) AS charge_energy_added,
			COALESCE(charge_energy_used, 0) AS charge_energy_used,
			COALESCE(cost, 0) AS cost,
			start_ideal_range_km AS start_ideal_range,
			(SELECT ideal_battery_range_km FROM charges WHERE charging_process_id = charging_processes.id ORDER BY id DESC LIMIT 1) AS current_ideal_range,
			start_rated_range_km AS start_rated_range,
			(SELECT rated_battery_range_km FROM charges WHERE charging_process_id = charging_processes.id ORDER BY id DESC LIMIT 1) AS current_rated_range,
			start_battery_level,
			(SELECT battery_level FROM charges WHERE charging_process_id = charging_processes.id ORDER BY id DESC LIMIT 1) AS current_battery_level,
			EXTRACT(EPOCH FROM (COALESCE(end_date, NOW()) - start_date))/60 AS duration_min,
			TO_CHAR((EXTRACT(EPOCH FROM (COALESCE(end_date, NOW()) - start_date))/60 * INTERVAL '1 minute'), 'HH24:MI') as duration_str,
			outside_temp_avg,
			position.odometer as odometer,
			(SELECT unit_of_length FROM settings LIMIT 1) as unit_of_length,
			(SELECT unit_of_temperature FROM settings LIMIT 1) as unit_of_temperature,
			cars.name,
			end_date IS NULL AS is_charging
		FROM charging_processes
		LEFT JOIN cars ON car_id = cars.id
		LEFT JOIN addresses address ON address_id = address.id
		LEFT JOIN positions position ON position_id = position.id
		LEFT JOIN geofences geofence ON geofence_id = geofence.id
		WHERE charging_processes.car_id=$1
		ORDER BY end_date IS NULL DESC, start_date DESC
		LIMIT 1;`

	row := db.QueryRow(query, CarID)

	// Scanning row and putting values into the temp vars to handle NULLs
	err := row.Scan(
		&charge.ChargeID,
		&charge.StartDate,
		&endDate,
		&address,
		&chargeEnergyAdded,
		&chargeEnergyUsed,
		&cost,
		&startIdealRange,
		&currentIdealRange,
		&startRatedRange,
		&currentRatedRange,
		&startBatteryLevel,
		&currentBatteryLevel,
		&durationMin,
		&durationStr,
		&outsideTempAvg,
		&odometer,
		&UnitsLength,
		&UnitsTemperature,
		&CarName,
		&isCharging,
	)

	switch err {
	case sql.ErrNoRows:
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsChargesCurrentV1", "No current charge found.", "No rows were returned")
		return
	case nil:
		// nothing wrong.. continuing
		break
	default:
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsChargesCurrentV1", CarsChargesCurrentError1, err.Error())
		return
	}

	// Set IsCharging and EndDate
	charge.IsCharging = isCharging
	if endDate.Valid {
		charge.EndDate = getTimeInTimeZone(endDate.String)
	} else {
		charge.EndDate = ""
	}

	// Handle NULLs in the database
	if address.Valid {
		charge.Address = address.String
	} else {
		charge.Address = "Unknown"
	}

	if chargeEnergyAdded.Valid {
		charge.ChargeEnergyAdded = chargeEnergyAdded.Float64
	}

	if chargeEnergyUsed.Valid {
		charge.ChargeEnergyUsed = chargeEnergyUsed.Float64
	}

	if cost.Valid {
		charge.Cost = cost.Float64
	}

	if startIdealRange.Valid {
		charge.RangeIdeal.StartRange = startIdealRange.Float64
	}

	if currentIdealRange.Valid {
		charge.RangeIdeal.CurrentRange = currentIdealRange.Float64
	}

	if startRatedRange.Valid {
		charge.RangeRated.StartRange = startRatedRange.Float64
	}

	if currentRatedRange.Valid {
		charge.RangeRated.CurrentRange = currentRatedRange.Float64
	}

	if startBatteryLevel.Valid {
		charge.BatteryDetails.StartBatteryLevel = int(startBatteryLevel.Int64)
	}

	if currentBatteryLevel.Valid {
		charge.BatteryDetails.CurrentBatteryLevel = int(currentBatteryLevel.Int64)
	}

	if durationMin.Valid {
		charge.DurationMin = int(durationMin.Float64) // Convert float64 to int
	}

	if durationStr.Valid {
		charge.DurationStr = durationStr.String
	}

	if outsideTempAvg.Valid {
		charge.OutsideTempAvg = outsideTempAvg.Float64
	}

	if odometer.Valid {
		charge.Odometer = odometer.Float64
	}

	// Converting values based on settings UnitsLength
	if UnitsLength == "mi" {
		charge.RangeIdeal.StartRange = kilometersToMiles(charge.RangeIdeal.StartRange)
		charge.RangeIdeal.CurrentRange = kilometersToMiles(charge.RangeIdeal.CurrentRange)
		charge.RangeRated.StartRange = kilometersToMiles(charge.RangeRated.StartRange)
		charge.RangeRated.CurrentRange = kilometersToMiles(charge.RangeRated.CurrentRange)
		charge.Odometer = kilometersToMiles(charge.Odometer)
	}
	// Converting values based on settings UnitsTemperature
	if UnitsTemperature == "F" && outsideTempAvg.Valid {
		charge.OutsideTempAvg = celsiusToFahrenheit(charge.OutsideTempAvg)
	}

	// Adjusting to timezone differences from UTC to be user-specific
	charge.StartDate = getTimeInTimeZone(charge.StartDate)

	// Getting detailed charge data from database
	query = `
		SELECT
			id AS detail_id,
			date,
			battery_level,
			usable_battery_level,
			charge_energy_added,
			not_enough_power_to_heat,
			COALESCE(charger_actual_current, 0) as charger_actual_current,
			COALESCE(charger_phases, 0) AS charger_phases,
			COALESCE(charger_pilot_current, 0) as charger_pilot_current,
			COALESCE(charger_power, 0) as charger_power,
			COALESCE(charger_voltage, 0) as charger_voltage,
			ideal_battery_range_km AS ideal_battery_range,
			rated_battery_range_km AS rated_battery_range,
			battery_heater,
			battery_heater_on,
			battery_heater_no_power,
			conn_charge_cable,
			fast_charger_present,
			fast_charger_brand,
			fast_charger_type,
			outside_temp
		FROM charges
		WHERE charging_process_id=$1
		ORDER BY id DESC
		LIMIT 50;`
	rows, err := db.Query(query, charge.ChargeID)

	// Checking for errors in query
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsChargesCurrentV1", CarsChargesCurrentError2, err.Error())
		return
	}

	// Defer closing rows
	defer rows.Close()

	// Looping through all results
	for rows.Next() {
		// Create temp variables to handle NULL values
		var (
			detailBatteryLevel, detailUsableBatteryLevel                                                 sql.NullInt64
			detailChargeEnergyAdded, detailIdealBatteryRange, detailRatedBatteryRange, detailOutsideTemp sql.NullFloat64
			detailConnChargeCable, detailFastChargerType                                                 sql.NullString
			detailFastChargerBrand                                                                       sql.NullString
		)

		// Creating chargedetails object based on struct
		chargedetails := ChargeDetails{}

		// Scanning row and putting values into temporary variables
		err = rows.Scan(
			&chargedetails.DetailID,
			&chargedetails.Date,
			&detailBatteryLevel,
			&detailUsableBatteryLevel,
			&detailChargeEnergyAdded,
			&chargedetails.NotEnoughPowerToHeat,
			&chargedetails.ChargerDetails.ChargerActualCurrent,
			&chargedetails.ChargerDetails.ChargerPhases,
			&chargedetails.ChargerDetails.ChargerPilotCurrent,
			&chargedetails.ChargerDetails.ChargerPower,
			&chargedetails.ChargerDetails.ChargerVoltage,
			&detailIdealBatteryRange,
			&detailRatedBatteryRange,
			&chargedetails.BatteryInfo.BatteryHeater,
			&chargedetails.BatteryInfo.BatteryHeaterOn,
			&chargedetails.BatteryInfo.BatteryHeaterNoPower,
			&detailConnChargeCable,
			&chargedetails.FastChargerInfo.FastChargerPresent,
			&detailFastChargerBrand,
			&detailFastChargerType,
			&detailOutsideTemp,
		)

		// Handle NULL values
		if detailBatteryLevel.Valid {
			chargedetails.BatteryLevel = int(detailBatteryLevel.Int64)
		}

		if detailUsableBatteryLevel.Valid {
			chargedetails.UsableBatteryLevel = int(detailUsableBatteryLevel.Int64)
		}

		if detailChargeEnergyAdded.Valid {
			chargedetails.ChargeEnergyAdded = detailChargeEnergyAdded.Float64
		}

		if detailIdealBatteryRange.Valid {
			chargedetails.BatteryInfo.IdealBatteryRange = detailIdealBatteryRange.Float64
		}

		if detailRatedBatteryRange.Valid {
			chargedetails.BatteryInfo.RatedBatteryRange = detailRatedBatteryRange.Float64
		}

		if detailConnChargeCable.Valid {
			chargedetails.ConnChargeCable = detailConnChargeCable.String
		}

		if detailFastChargerBrand.Valid {
			chargedetails.FastChargerInfo.FastChargerBrand.String = detailFastChargerBrand.String
			chargedetails.FastChargerInfo.FastChargerBrand.Valid = true
		}

		if detailFastChargerType.Valid {
			chargedetails.FastChargerInfo.FastChargerType = detailFastChargerType.String
		}

		if detailOutsideTemp.Valid {
			chargedetails.OutsideTemp = detailOutsideTemp.Float64
		}

		// Converting values based on settings UnitsLength
		if UnitsLength == "mi" && detailIdealBatteryRange.Valid {
			chargedetails.BatteryInfo.IdealBatteryRange = kilometersToMiles(chargedetails.BatteryInfo.IdealBatteryRange)
		}

		if UnitsLength == "mi" && detailRatedBatteryRange.Valid {
			chargedetails.BatteryInfo.RatedBatteryRange = kilometersToMiles(chargedetails.BatteryInfo.RatedBatteryRange)
		}

		// Converting values based on settings UnitsTemperature
		if UnitsTemperature == "F" && detailOutsideTemp.Valid {
			chargedetails.OutsideTemp = celsiusToFahrenheit(chargedetails.OutsideTemp)
		}

		// Adjusting to timezone differences from UTC to be user-specific
		chargedetails.Date = getTimeInTimeZone(chargedetails.Date)

		// Checking for errors after scanning
		if err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsChargesCurrentV1", CarsChargesCurrentError2, err.Error())
			return
		}

		// Appending chargedetails to ChargeDetailsData
		ChargeDetailsData = append(ChargeDetailsData, chargedetails)
	}

	// Set the ChargeDetails in the charge
	charge.ChargeDetails = ChargeDetailsData

	// Checking for errors in the rows result
	err = rows.Err()
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsChargesCurrentV1", CarsChargesCurrentError2, err.Error())
		return
	}

	// Build the data-blob
	jsonData := JSONData{
		Data{
			Car: Car{
				CarID:   CarID,
				CarName: CarName,
			},
			Charge: charge,
			TeslaMateUnits: TeslaMateUnits{
				UnitsLength:      UnitsLength,
				UnitsTemperature: UnitsTemperature,
			},
		},
	}

	// Return jsonData
	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsChargesCurrentV1", jsonData)
}
