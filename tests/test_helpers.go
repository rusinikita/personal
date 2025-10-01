package tests

import (
	"math/rand"

	"personal/domain"
	"personal/util"
)

// GenerateRandomFoodName generates a random food name for testing
func GenerateRandomFoodName(rng *rand.Rand) string {
	foods := []string{
		"Apple", "Banana", "Orange", "Chicken Breast", "Salmon",
		"Rice", "Pasta", "Bread", "Egg", "Milk",
		"Cheese", "Yogurt", "Beef", "Pork", "Turkey",
		"Potato", "Carrot", "Broccoli", "Spinach", "Tomato",
		"Cucumber", "Lettuce", "Onion", "Garlic", "Pepper",
		"Strawberry", "Blueberry", "Grape", "Watermelon", "Mango",
		"Pineapple", "Avocado", "Almond", "Walnut", "Cashew",
		"Oatmeal", "Quinoa", "Lentil", "Chickpea", "Tofu",
	}
	return foods[rng.Intn(len(foods))]
}

// GenerateRandomServingName generates a random serving name for testing
func GenerateRandomServingName(rng *rand.Rand) string {
	servings := []string{
		"piece", "slice", "cup", "serving", "cookie",
		"bar", "portion", "scoop", "spoon", "glass",
	}
	return servings[rng.Intn(len(servings))]
}

// GenerateRandomNutrients generates random nutrients for testing
func GenerateRandomNutrients(rng *rand.Rand) *domain.Nutrients {
	return &domain.Nutrients{
		Calories:       util.Ptr(rng.Float64() * 500), // 0-500 calories
		ProteinG:       util.Ptr(rng.Float64() * 50),  // 0-50g protein
		TotalFatG:      util.Ptr(rng.Float64() * 30),  // 0-30g fat
		CarbohydratesG: util.Ptr(rng.Float64() * 100), // 0-100g carbs
		DietaryFiberG:  util.Ptr(rng.Float64() * 10),  // 0-10g fiber
		TotalSugarsG:   util.Ptr(rng.Float64() * 20),  // 0-20g sugar
	}
}
