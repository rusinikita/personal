package tests

import (
	"context"
	"math/rand"
	"sort"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/domain"
	"personal/util"
)

func (s *IntegrationTestSuite) TestGetTopProducts_Success() {
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

	// Define time window: last 2 months (within 3 months window)
	twoMonthsAgo := now.AddDate(0, -2, 0)

	// Random generator
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Generate 100-200 random records
	numRecords := rng.Intn(101) + 100 // 100-200

	// We want 40 different food_id with different frequencies
	// Let's distribute records across food_ids with decreasing frequency
	foodIDFrequencies := make(map[int64]int)
	remainingRecords := numRecords

	// Assign frequencies to 40 food_ids (descending)
	for foodID := int64(1); foodID <= 40 && remainingRecords > 0; foodID++ {
		// Higher food_ids get fewer records
		// First food_id gets more, last gets fewer
		frequency := remainingRecords / int(41-foodID)
		if frequency == 0 {
			frequency = 1
		}
		foodIDFrequencies[foodID] = frequency
		remainingRecords -= frequency
	}

	// Distribute any remaining records randomly
	for remainingRecords > 0 {
		foodID := int64(rng.Intn(40) + 1)
		foodIDFrequencies[foodID]++
		remainingRecords--
	}

	// Create food records in food table first
	foodRecords := make(map[int64]*domain.Food)
	for foodID := int64(1); foodID <= 40; foodID++ {
		food := &domain.Food{
			ID:          util.Ptr(foodID),
			Name:        util.Ptr(GenerateRandomFoodName(rng)),
			Description: nil,
			Barcode:     nil,
			FoodType:    util.Ptr("product"),
			IsArchived:  util.Ptr(false),
			Nutrients:   GenerateRandomNutrients(rng),
		}

		// Some foods have serving_name
		if rng.Float64() < 0.5 {
			food.ServingName = util.Ptr(GenerateRandomServingName(rng))
		}

		err := s.Repo().CreateFood(ctx, food)
		require.NoError(s.T(), err)

		foodRecords[foodID] = food
	}

	// Generate consumption log records
	usedTimestamps := make(map[time.Time]bool)
	var allRecords []*domain.ConsumptionLog

	for foodID, frequency := range foodIDFrequencies {
		for i := 0; i < frequency; i++ {
			// Generate random time within the time window (last 2 months)
			timeRange := now.Sub(twoMonthsAgo)
			randomDuration := time.Duration(rng.Int63n(int64(timeRange)))
			consumedAt := twoMonthsAgo.Add(randomDuration)

			// Ensure timestamp is unique
			for usedTimestamps[consumedAt] {
				consumedAt = consumedAt.Add(time.Microsecond)
			}
			usedTimestamps[consumedAt] = true

			// Random nutrients and amount
			weight := rng.Float64()*200 + 50 // 50-250g weight

			record := &domain.ConsumptionLog{
				UserID:     userID,
				ConsumedAt: consumedAt,
				FoodID:     util.Ptr(foodID),
				FoodName:   *foodRecords[foodID].Name,
				AmountG:    weight,
				Nutrients:  GenerateRandomNutrients(rng),
			}

			allRecords = append(allRecords, record)
		}
	}

	// Save all records to database
	for _, record := range allRecords {
		err := s.Repo().AddConsumptionLog(ctx, record)
		require.NoError(s.T(), err)
	}

	// Calculate expected top 30 products
	type foodStat struct {
		foodID      int64
		foodName    string
		servingName string
		logCount    int
	}

	foodStats := make([]foodStat, 0)
	for foodID, frequency := range foodIDFrequencies {
		food := foodRecords[foodID]
		servingName := ""
		if food.ServingName != nil {
			servingName = *food.ServingName
		}

		foodStats = append(foodStats, foodStat{
			foodID:      foodID,
			foodName:    *food.Name,
			servingName: servingName,
			logCount:    frequency,
		})
	}

	// Sort by log_count DESC, food_id ASC
	sort.Slice(foodStats, func(i, j int) bool {
		if foodStats[i].logCount != foodStats[j].logCount {
			return foodStats[i].logCount > foodStats[j].logCount
		}
		return foodStats[i].foodID < foodStats[j].foodID
	})

	// Take top 30
	expectedTop30 := foodStats
	if len(expectedTop30) > 30 {
		expectedTop30 = foodStats[:30]
	}

	// TODO: Call MCP get_top_products tool handler
	// _, output, err := top_products.GetTopProducts(s.ContextWithDB(ctx), nil, struct{}{})
	// require.NoError(s.T(), err)

	// TODO: Verify results once handler is implemented
	// require.Len(s.T(), output.Products, len(expectedTop30))
	//
	// for i, expected := range expectedTop30 {
	// 	actual := output.Products[i]
	// 	assert.Equal(s.T(), expected.foodID, actual.FoodID)
	// 	assert.Equal(s.T(), expected.foodName, actual.FoodName)
	// 	assert.Equal(s.T(), expected.servingName, actual.ServingName)
	// 	assert.Equal(s.T(), expected.logCount, actual.LogCount)
	// }

	// Verify at least that we can call repository method
	threeMonthsAgo := now.AddDate(0, -3, 0)
	topProducts, err := s.Repo().GetTopProducts(ctx, userID, threeMonthsAgo, now, 30)
	require.NoError(s.T(), err)
	require.Len(s.T(), topProducts, len(expectedTop30))

	for i, expected := range expectedTop30 {
		actual := topProducts[i]
		assert.Equal(s.T(), expected.foodID, actual.FoodID)
		assert.Equal(s.T(), expected.foodName, actual.FoodName)
		assert.Equal(s.T(), expected.servingName, actual.ServingName)
		assert.Equal(s.T(), expected.logCount, actual.LogCount)
	}

	// Cleanup
	for _, record := range allRecords {
		defer s.Repo().DeleteConsumptionLog(ctx, userID, record.ConsumedAt)
	}

	for foodID := int64(1); foodID <= 40; foodID++ {
		defer s.Repo().DeleteFood(ctx, foodID)
	}
}
