package tests

import (
	"context"
	"math/rand"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/nutrition_stats"
	"personal/domain"
	"personal/util"
)

func (s *IntegrationTestSuite) TestGetNutritionStats_Success() {
	ctx := context.Background()
	userID := int64(1)

	// Clean up any existing data for this user
	existingLogs, _ := s.Repo().GetConsumptionLogsByUser(ctx, userID)
	for _, log := range existingLogs {
		_ = s.Repo().DeleteConsumptionLog(ctx, userID, log.ConsumedAt)
	}

	// Use UTC timezone
	location := time.UTC

	// Get current time in UTC
	now := time.Now().UTC()

	// Define 3 test days: day before yesterday, yesterday, today
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	yesterday := today.AddDate(0, 0, -1)
	dayBeforeYesterday := today.AddDate(0, 0, -2)

	days := []time.Time{dayBeforeYesterday, yesterday, today}

	// Random generator
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Expected aggregations
	expectedByDay := make(map[string]*domain.NutritionStats)
	var allRecords []*domain.ConsumptionLog
	var lastRecordTime time.Time
	usedTimestamps := make(map[time.Time]bool)

	// Generate at least one record per day to ensure all 3 days are represented
	for _, day := range days {
		// Generate random time within the day
		var consumedAt time.Time
		if day.Equal(today) {
			// For today, ensure time is before now
			maxHour := now.Hour()
			if maxHour == 0 {
				maxHour = 1 // Avoid division by zero
			}
			hour := rng.Intn(maxHour)
			minute := rng.Intn(60)
			second := rng.Intn(60)
			ns := rng.Intn(1000000000)
			consumedAt = time.Date(day.Year(), day.Month(), day.Day(), hour, minute, second, ns, location)
		} else {
			// For past days, use any time within the day
			hour := rng.Intn(24)
			minute := rng.Intn(60)
			second := rng.Intn(60)
			ns := rng.Intn(1000000000)
			consumedAt = time.Date(day.Year(), day.Month(), day.Day(), hour, minute, second, ns, location)
		}

		// Random nutrients and amount
		calories := rng.Float64() * 500  // 0-500 calories
		protein := rng.Float64() * 50    // 0-50g protein
		fat := rng.Float64() * 30        // 0-30g fat
		carbs := rng.Float64() * 100     // 0-100g carbs
		weight := rng.Float64()*200 + 50 // 50-250g weight

		// Ensure timestamp is unique
		for usedTimestamps[consumedAt] {
			consumedAt = consumedAt.Add(time.Microsecond)
		}
		usedTimestamps[consumedAt] = true

		record := &domain.ConsumptionLog{
			UserID:     userID,
			ConsumedAt: consumedAt,
			FoodID:     nil,
			FoodName:   "Test Food",
			AmountG:    weight,
			Nutrients: &domain.Nutrients{
				Calories:       util.Ptr(calories),
				ProteinG:       util.Ptr(protein),
				TotalFatG:      util.Ptr(fat),
				CarbohydratesG: util.Ptr(carbs),
			},
		}

		allRecords = append(allRecords, record)

		// Track last record time
		if consumedAt.After(lastRecordTime) {
			lastRecordTime = consumedAt
		}

		// Aggregate by day
		dayKey := day.Format("2006-01-02")
		if expectedByDay[dayKey] == nil {
			expectedByDay[dayKey] = &domain.NutritionStats{
				PeriodStart: time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, location),
				PeriodEnd:   time.Date(day.Year(), day.Month(), day.Day(), 23, 59, 59, 0, location),
			}
		}
		expectedByDay[dayKey].TotalCalories += calories
		expectedByDay[dayKey].TotalProtein += protein
		expectedByDay[dayKey].TotalFat += fat
		expectedByDay[dayKey].TotalCarbs += carbs
		expectedByDay[dayKey].TotalWeight += weight
	}

	// Generate additional random records
	additionalRecords := rng.Intn(8) + 2 // 2-9 additional records
	for i := 0; i < additionalRecords; i++ {
		// Pick random day
		day := days[rng.Intn(len(days))]

		// Generate random time within the day
		var consumedAt time.Time
		if day.Equal(today) {
			// For today, ensure time is before now
			maxHour := now.Hour()
			if maxHour == 0 {
				maxHour = 1
			}
			hour := rng.Intn(maxHour)
			minute := rng.Intn(60)
			second := rng.Intn(60)
			consumedAt = time.Date(day.Year(), day.Month(), day.Day(), hour, minute, second, 0, location)
		} else {
			hour := rng.Intn(24)
			minute := rng.Intn(60)
			second := rng.Intn(60)
			consumedAt = time.Date(day.Year(), day.Month(), day.Day(), hour, minute, second, 0, location)
		}

		// Random nutrients and amount
		calories := rng.Float64() * 500  // 0-500 calories
		protein := rng.Float64() * 50    // 0-50g protein
		fat := rng.Float64() * 30        // 0-30g fat
		carbs := rng.Float64() * 100     // 0-100g carbs
		weight := rng.Float64()*200 + 50 // 50-250g weight

		// Ensure timestamp is unique
		for usedTimestamps[consumedAt] {
			consumedAt = consumedAt.Add(time.Microsecond)
		}
		usedTimestamps[consumedAt] = true

		record := &domain.ConsumptionLog{
			UserID:     userID,
			ConsumedAt: consumedAt,
			FoodID:     nil,
			FoodName:   "Test Food",
			AmountG:    weight,
			Nutrients: &domain.Nutrients{
				Calories:       util.Ptr(calories),
				ProteinG:       util.Ptr(protein),
				TotalFatG:      util.Ptr(fat),
				CarbohydratesG: util.Ptr(carbs),
			},
		}

		allRecords = append(allRecords, record)

		// Track last record time
		if consumedAt.After(lastRecordTime) {
			lastRecordTime = consumedAt
		}

		// Aggregate by day
		dayKey := day.Format("2006-01-02")
		if expectedByDay[dayKey] == nil {
			expectedByDay[dayKey] = &domain.NutritionStats{
				PeriodStart: time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, location),
				PeriodEnd:   time.Date(day.Year(), day.Month(), day.Day(), 23, 59, 59, 0, location),
			}
		}
		expectedByDay[dayKey].TotalCalories += calories
		expectedByDay[dayKey].TotalProtein += protein
		expectedByDay[dayKey].TotalFat += fat
		expectedByDay[dayKey].TotalCarbs += carbs
		expectedByDay[dayKey].TotalWeight += weight
	}

	// Save all records to database
	for _, record := range allRecords {
		err := s.Repo().AddConsumptionLog(ctx, record)
		require.NoError(s.T(), err)
	}

	// Calculate expected last meal (1 hour before and including last record)
	expectedLastMeal := &domain.NutritionStats{
		PeriodStart: lastRecordTime.Add(-1 * time.Hour),
		PeriodEnd:   lastRecordTime,
	}
	hourBeforeLast := lastRecordTime.Add(-1 * time.Hour)
	for _, record := range allRecords {
		// Include records where: hourBeforeLast <= consumed_at <= lastRecordTime
		if (record.ConsumedAt.Equal(hourBeforeLast) || record.ConsumedAt.After(hourBeforeLast)) &&
			(record.ConsumedAt.Equal(lastRecordTime) || record.ConsumedAt.Before(lastRecordTime)) {
			if record.Nutrients.Calories != nil {
				expectedLastMeal.TotalCalories += *record.Nutrients.Calories
			}
			if record.Nutrients.ProteinG != nil {
				expectedLastMeal.TotalProtein += *record.Nutrients.ProteinG
			}
			if record.Nutrients.TotalFatG != nil {
				expectedLastMeal.TotalFat += *record.Nutrients.TotalFatG
			}
			if record.Nutrients.CarbohydratesG != nil {
				expectedLastMeal.TotalCarbs += *record.Nutrients.CarbohydratesG
			}
			expectedLastMeal.TotalWeight += record.AmountG
		}
	}

	// Build expected last 4 days array (chronologically sorted)
	var expectedLast4Days []domain.NutritionStats
	for _, day := range days {
		dayKey := day.Format("2006-01-02")
		if stats, exists := expectedByDay[dayKey]; exists {
			expectedLast4Days = append(expectedLast4Days, *stats)
		}
	}

	// Call MCP get_nutrition_stats tool handler
	_, output, err := nutrition_stats.GetNutritionStats(s.ContextWithDB(ctx), nil, struct{}{})
	require.NoError(s.T(), err)

	// Verify last_meal
	assert.InDelta(s.T(), expectedLastMeal.TotalCalories, output.LastMeal.TotalCalories, 0.01)
	assert.InDelta(s.T(), expectedLastMeal.TotalProtein, output.LastMeal.TotalProtein, 0.01)
	assert.InDelta(s.T(), expectedLastMeal.TotalFat, output.LastMeal.TotalFat, 0.01)
	assert.InDelta(s.T(), expectedLastMeal.TotalCarbs, output.LastMeal.TotalCarbs, 0.01)
	assert.InDelta(s.T(), expectedLastMeal.TotalWeight, output.LastMeal.TotalWeight, 0.01)

	// Verify last_4_days
	require.Len(s.T(), output.Last4Days, len(expectedLast4Days))
	for i, expected := range expectedLast4Days {
		actual := output.Last4Days[i]
		assert.True(s.T(), expected.PeriodStart.Equal(actual.PeriodStart), "PeriodStart mismatch: expected %v, got %v", expected.PeriodStart, actual.PeriodStart)
		assert.True(s.T(), expected.PeriodEnd.Equal(actual.PeriodEnd), "PeriodEnd mismatch: expected %v, got %v", expected.PeriodEnd, actual.PeriodEnd)
		assert.InDelta(s.T(), expected.TotalCalories, actual.TotalCalories, 0.01)
		assert.InDelta(s.T(), expected.TotalProtein, actual.TotalProtein, 0.01)
		assert.InDelta(s.T(), expected.TotalFat, actual.TotalFat, 0.01)
		assert.InDelta(s.T(), expected.TotalCarbs, actual.TotalCarbs, 0.01)
		assert.InDelta(s.T(), expected.TotalWeight, actual.TotalWeight, 0.01)
	}

	// Cleanup
	for _, record := range allRecords {
		defer s.Repo().DeleteConsumptionLog(ctx, userID, record.ConsumedAt)
	}
}

func (s *IntegrationTestSuite) TestGetNutritionStats_EmptyDatabase() {
	ctx := context.Background()

	// Call MCP get_nutrition_stats tool handler
	_, output, err := nutrition_stats.GetNutritionStats(s.ContextWithDB(ctx), nil, struct{}{})
	require.NoError(s.T(), err)

	// Verify empty results
	assert.Equal(s.T(), 0.0, output.LastMeal.TotalCalories)
	assert.Equal(s.T(), 0.0, output.LastMeal.TotalProtein)
	assert.Equal(s.T(), 0.0, output.LastMeal.TotalFat)
	assert.Equal(s.T(), 0.0, output.LastMeal.TotalCarbs)
	assert.Equal(s.T(), 0.0, output.LastMeal.TotalWeight)
	assert.Empty(s.T(), output.Last4Days)
}

func (s *IntegrationTestSuite) TestGetNutritionStats_TimezoneBoundaries() {
	ctx := context.Background()
	userID := int64(1)

	// Clean up any existing data for this user
	existingLogs, _ := s.Repo().GetConsumptionLogsByUser(ctx, userID)
	for _, log := range existingLogs {
		_ = s.Repo().DeleteConsumptionLog(ctx, userID, log.ConsumedAt)
	}

	// Use UTC timezone
	location := time.UTC

	// Get current date and time in UTC
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	yesterday := today.AddDate(0, 0, -1)

	// Create records at day boundaries
	// 23:59 yesterday should go to yesterday
	yesterdayEnd := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 23, 59, 0, 0, location)
	// Use current time for today (ensures it's before "now")
	todayRecord := now.Add(-10 * time.Minute) // 10 minutes ago

	records := []*domain.ConsumptionLog{
		{
			UserID:     userID,
			ConsumedAt: yesterdayEnd,
			FoodName:   "Yesterday Record",
			AmountG:    100.0,
			Nutrients: &domain.Nutrients{
				Calories:       util.Ptr(200.0),
				ProteinG:       util.Ptr(10.0),
				TotalFatG:      util.Ptr(5.0),
				CarbohydratesG: util.Ptr(25.0),
			},
		},
		{
			UserID:     userID,
			ConsumedAt: todayRecord,
			FoodName:   "Today Record",
			AmountG:    150.0,
			Nutrients: &domain.Nutrients{
				Calories:       util.Ptr(300.0),
				ProteinG:       util.Ptr(15.0),
				TotalFatG:      util.Ptr(8.0),
				CarbohydratesG: util.Ptr(40.0),
			},
		},
	}

	// Save records
	for _, record := range records {
		err := s.Repo().AddConsumptionLog(ctx, record)
		require.NoError(s.T(), err)
	}

	// Call MCP get_nutrition_stats tool handler
	_, output, err := nutrition_stats.GetNutritionStats(s.ContextWithDB(ctx), nil, struct{}{})
	require.NoError(s.T(), err)

	// Verify that records are in correct day buckets
	require.Len(s.T(), output.Last4Days, 2) // Yesterday and today

	// Yesterday stats
	yesterdayStats := output.Last4Days[0]
	assert.Equal(s.T(), 200.0, yesterdayStats.TotalCalories)

	// Today stats
	todayStats := output.Last4Days[1]
	assert.Equal(s.T(), 300.0, todayStats.TotalCalories)

	// Cleanup
	for _, record := range records {
		defer s.Repo().DeleteConsumptionLog(ctx, userID, record.ConsumedAt)
	}
}
