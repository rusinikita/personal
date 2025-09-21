package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type Food struct {
	ID              int64             `json:"id" db:"id"`
	Name            string            `json:"name" db:"name"`
	Description     *string           `json:"description,omitempty" db:"description"`
	Barcode         *string           `json:"barcode,omitempty" db:"barcode"`
	FoodType        string            `json:"food_type" db:"food_type"`
	IsArchived      bool              `json:"is_archived" db:"is_archived"`
	ServingSizeG    *float64          `json:"serving_size_g,omitempty" db:"serving_size_g"`
	ServingName     *string           `json:"serving_name,omitempty" db:"serving_name"`
	CreatedAt       time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at" db:"updated_at"`
	Nutrients       *Nutrients        `json:"nutrients,omitempty" db:"nutrients"`
	FoodComposition FoodComponentList `json:"food_composition,omitempty" db:"food_composition"`
}

type Nutrients struct {
	// Макронутриенты (на 100г продукта)
	Calories       *float64 `json:"calories,omitempty"`
	ProteinG       *float64 `json:"protein_g,omitempty"`
	TotalFatG      *float64 `json:"total_fat_g,omitempty"`
	CarbohydratesG *float64 `json:"carbohydrates_g,omitempty"`
	DietaryFiberG  *float64 `json:"dietary_fiber_g,omitempty"`
	TotalSugarsG   *float64 `json:"total_sugars_g,omitempty"`
	AddedSugarsG   *float64 `json:"added_sugars_g,omitempty"`
	WaterG         *float64 `json:"water_g,omitempty"`

	// Детализация жиров (в граммах)
	SaturatedFatsG       *float64 `json:"saturated_fats_g,omitempty"`
	MonounsaturatedFatsG *float64 `json:"monounsaturated_fats_g,omitempty"`
	PolyunsaturatedFatsG *float64 `json:"polyunsaturated_fats_g,omitempty"`
	TransFatsG           *float64 `json:"trans_fats_g,omitempty"`

	// Омега жирные кислоты (в миллиграммах)
	Omega3Mg               *float64 `json:"omega_3_mg,omitempty"`
	Omega6Mg               *float64 `json:"omega_6_mg,omitempty"`
	Omega9Mg               *float64 `json:"omega_9_mg,omitempty"`
	AlphaLinolenicAcidMg   *float64 `json:"alpha_linolenic_acid_mg,omitempty"`
	LinoleicAcidMg         *float64 `json:"linoleic_acid_mg,omitempty"`
	EicosapentaenoicAcidMg *float64 `json:"eicosapentaenoic_acid_mg,omitempty"`
	DocosahexaenoicAcidMg  *float64 `json:"docosahexaenoic_acid_mg,omitempty"`

	// Холестерин (в миллиграммах)
	CholesterolMg *float64 `json:"cholesterol_mg,omitempty"`

	// Витамины
	VitaminAMcg   *float64 `json:"vitamin_a_mcg,omitempty"`
	VitaminCMg    *float64 `json:"vitamin_c_mg,omitempty"`
	VitaminDMcg   *float64 `json:"vitamin_d_mcg,omitempty"`
	VitaminEMg    *float64 `json:"vitamin_e_mg,omitempty"`
	VitaminKMcg   *float64 `json:"vitamin_k_mcg,omitempty"`
	VitaminB1Mg   *float64 `json:"vitamin_b1_mg,omitempty"`
	VitaminB2Mg   *float64 `json:"vitamin_b2_mg,omitempty"`
	VitaminB3Mg   *float64 `json:"vitamin_b3_mg,omitempty"`
	VitaminB5Mg   *float64 `json:"vitamin_b5_mg,omitempty"`
	VitaminB6Mg   *float64 `json:"vitamin_b6_mg,omitempty"`
	VitaminB7Mcg  *float64 `json:"vitamin_b7_mcg,omitempty"`
	VitaminB9Mcg  *float64 `json:"vitamin_b9_mcg,omitempty"`
	VitaminB12Mcg *float64 `json:"vitamin_b12_mcg,omitempty"`
	FolateDfeMcg  *float64 `json:"folate_dfe_mcg,omitempty"`
	CholineMg     *float64 `json:"choline_mg,omitempty"`

	// Минералы (в миллиграммах)
	CalciumMg    *float64 `json:"calcium_mg,omitempty"`
	IronMg       *float64 `json:"iron_mg,omitempty"`
	MagnesiumMg  *float64 `json:"magnesium_mg,omitempty"`
	PhosphorusMg *float64 `json:"phosphorus_mg,omitempty"`
	PotassiumMg  *float64 `json:"potassium_mg,omitempty"`
	SodiumMg     *float64 `json:"sodium_mg,omitempty"`
	ZincMg       *float64 `json:"zinc_mg,omitempty"`
	CopperMg     *float64 `json:"copper_mg,omitempty"`
	ManganeseMg  *float64 `json:"manganese_mg,omitempty"`
	SeleniumMcg  *float64 `json:"selenium_mcg,omitempty"`
	IodineMcg    *float64 `json:"iodine_mcg,omitempty"`

	// Аминокислоты (в миллиграммах)
	LysineMg        *float64 `json:"lysine_mg,omitempty"`
	MethionineMg    *float64 `json:"methionine_mg,omitempty"`
	CysteineMg      *float64 `json:"cysteine_mg,omitempty"`
	PhenylalanineMg *float64 `json:"phenylalanine_mg,omitempty"`
	TyrosineMg      *float64 `json:"tyrosine_mg,omitempty"`
	ThreonineMg     *float64 `json:"threonine_mg,omitempty"`
	TryptophanMg    *float64 `json:"tryptophan_mg,omitempty"`
	ValineMg        *float64 `json:"valine_mg,omitempty"`
	HistidineMg     *float64 `json:"histidine_mg,omitempty"`
	LeucineMg       *float64 `json:"leucine_mg,omitempty"`
	IsoleucineMg    *float64 `json:"isoleucine_mg,omitempty"`

	// Специальные вещества
	CaffeineMg    *float64 `json:"caffeine_mg,omitempty"`
	EthylAlcoholG *float64 `json:"ethyl_alcohol_g,omitempty"`

	// Дополнительные поля
	GlycemicIndex *int     `json:"glycemic_index,omitempty"`
	GlycemicLoad  *float64 `json:"glycemic_load,omitempty"`
}

type FoodComponent struct {
	FoodID  int64   `json:"food_id"`
	AmountG float64 `json:"amount_g"`
}

type FoodComponentList []FoodComponent

// Value implements driver.Valuer for JSONB
func (f FoodComponentList) Value() (driver.Value, error) {
	if f == nil {
		return nil, nil
	}
	return json.Marshal(f)
}

// Scan implements sql.Scanner for JSONB
func (f *FoodComponentList) Scan(value interface{}) error {
	if value == nil {
		*f = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into FoodComponentList", value)
	}

	return json.Unmarshal(bytes, f)
}

// Value implements driver.Valuer for JSONB
func (n *Nutrients) Value() (driver.Value, error) {
	if n == nil {
		return nil, nil
	}
	return json.Marshal(n)
}

// Scan implements sql.Scanner for JSONB
func (n *Nutrients) Scan(value interface{}) error {
	if value == nil {
		*n = Nutrients{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into Nutrients", value)
	}

	return json.Unmarshal(bytes, n)
}
