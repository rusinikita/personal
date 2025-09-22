-- =====================================================
-- Система учета питания - SQL скрипт создания БД
-- =====================================================

-- =====================================================
-- FOOD - Основная таблица продуктов и блюд
-- =====================================================
CREATE TABLE IF NOT EXISTS food (
                    id BIGSERIAL PRIMARY KEY,
                    name VARCHAR(255) NOT NULL,
                    description TEXT,
                    barcode VARCHAR(50) UNIQUE,
                    food_type VARCHAR(20) NOT NULL, --('component', 'product', 'dish'))
                    is_archived BOOLEAN DEFAULT FALSE,
                    serving_size_g DECIMAL(8,2), -- Nullable, размер стандартной порции
                    serving_name varchar(20), -- Nullable, название порции (например, "cookie")
                    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

                    nutrients JSONB, -- JSON объект данных о составе
                    food_composition JSONB, -- JSON список пар food_id, amount_g для рецепта 'dish' состоящего из 'component' и 'product'

                    CONSTRAINT check_serving_positive CHECK (serving_size_g > 0 OR serving_size_g IS NULL)
    );

-- =====================================================
-- CONSUMPTION_LOG - Журнал потребления
-- =====================================================
CREATE TABLE IF NOT EXISTS consumption_log (
                    user_id BIGINT NOT NULL,
                    consumed_at TIMESTAMP NOT NULL,
                    food_id BIGINT, -- Nullable для сценария с direct_nutrients
                    food_name VARCHAR(255) NOT NULL, -- Название продукта
                    amount_g DECIMAL(8,2) NOT NULL CHECK (amount_g > 0),
                    meal_type VARCHAR(20), -- ('breakfast', 'lunch', 'dinner', 'snack', 'other')
                    note TEXT,

                    nutrients JSONB, -- Снимок всех нутриентов на момент потребления (те же что и в food)

                    PRIMARY KEY (user_id, consumed_at),
                    FOREIGN KEY (food_id) REFERENCES food(id)
    );

-- =====================================================
-- Состав внутри JSONB объекта nutrients
-- =====================================================
-- ===== МАКРОНУТРИЕНТЫ (на 100г продукта) =====
-- calories DECIMAL(7,2),           -- ккал
-- protein_g DECIMAL(6,2),          -- белки в граммах
-- total_fat_g DECIMAL(6,2),        -- общие жиры в граммах
-- carbohydrates_g DECIMAL(6,2),    -- углеводы в граммах
-- dietary_fiber_g DECIMAL(5,2),    -- клетчатка в граммах
-- total_sugars_g DECIMAL(5,2),     -- сахара в граммах
-- added_sugars_g DECIMAL(5,2),     -- добавленные сахара в граммах
-- water_g DECIMAL(5,2),            -- вода в граммах

-- ===== ДЕТАЛИЗАЦИЯ ЖИРОВ (в граммах) =====
-- saturated_fats_g DECIMAL(5,2),        -- насыщенные жиры
-- monounsaturated_fats_g DECIMAL(5,2),  -- мононенасыщенные
-- polyunsaturated_fats_g DECIMAL(5,2),  -- полиненасыщенные
-- trans_fats_g DECIMAL(5,2),            -- транс-жиры
--
-- ===== ОМЕГА ЖИРНЫЕ КИСЛОТЫ (в миллиграммах) =====
-- omega_3_mg DECIMAL(7,2),                   -- омега-3
-- omega_6_mg DECIMAL(7,2),                   -- омега-6
-- omega_9_mg DECIMAL(7,2),                   -- омега-9
-- alpha_linolenic_acid_mg DECIMAL(6,2),      -- альфа-линоленовая
-- linoleic_acid_mg DECIMAL(6,2),             -- линолевая
-- eicosapentaenoic_acid_mg DECIMAL(6,2),     -- EPA
-- docosahexaenoic_acid_mg DECIMAL(6,2),      -- DHA
--
-- ===== ХОЛЕСТЕРИН (в миллиграммах) =====
-- cholesterol_mg DECIMAL(6,2),
--
-- ===== ВИТАМИНЫ =====
-- vitamin_a_mcg DECIMAL(7,2),     -- микрограммы
-- vitamin_c_mg DECIMAL(6,2),      -- миллиграммы
-- vitamin_d_mcg DECIMAL(6,2),     -- микрограммы
-- vitamin_e_mg DECIMAL(6,2),      -- миллиграммы
-- vitamin_k_mcg DECIMAL(6,2),     -- микрограммы
-- vitamin_b1_mg DECIMAL(5,2),     -- тиамин
-- vitamin_b2_mg DECIMAL(5,2),     -- рибофлавин
-- vitamin_b3_mg DECIMAL(5,2),     -- ниацин
-- vitamin_b5_mg DECIMAL(5,2),     -- пантотеновая кислота
-- vitamin_b6_mg DECIMAL(5,2),
-- vitamin_b7_mcg DECIMAL(6,2),    -- биотин в микрограммах
-- vitamin_b9_mcg DECIMAL(6,2),    -- фолиевая кислота в микрограммах
-- vitamin_b12_mcg DECIMAL(5,2),   -- в микрограммах
-- folate_dfe_mcg DECIMAL(6,2),    -- фолат в микрограммах DFE
-- choline_mg DECIMAL(6,2),        -- холин в миллиграммах
--
-- ===== МИНЕРАЛЫ (в миллиграммах) =====
-- calcium_mg DECIMAL(6,2),
-- iron_mg DECIMAL(5,2),
-- magnesium_mg DECIMAL(6,2),
-- phosphorus_mg DECIMAL(6,2),
-- potassium_mg DECIMAL(6,2),
-- sodium_mg DECIMAL(6,2),
-- zinc_mg DECIMAL(5,2),
-- copper_mg DECIMAL(5,2),
-- manganese_mg DECIMAL(5,2),
-- selenium_mcg DECIMAL(6,2),      -- в микрограммах
-- iodine_mcg DECIMAL(6,2),        -- в микрограммах
--
-- ===== АМИНОКИСЛОТЫ (в миллиграммах) =====
-- lysine_mg DECIMAL(6,2),
-- methionine_mg DECIMAL(6,2),
-- cysteine_mg DECIMAL(6,2),
-- phenylalanine_mg DECIMAL(6,2),
-- tyrosine_mg DECIMAL(6,2),
-- threonine_mg DECIMAL(6,2),
-- tryptophan_mg DECIMAL(6,2),
-- valine_mg DECIMAL(6,2),
-- histidine_mg DECIMAL(6,2),
-- leucine_mg DECIMAL(6,2),
-- isoleucine_mg DECIMAL(6,2),
--
-- ===== СПЕЦИАЛЬНЫЕ ВЕЩЕСТВА =====
-- caffeine_mg DECIMAL(6,2),       -- кофеин в миллиграммах
-- ethyl_alcohol_g DECIMAL(5,2),   -- алкоголь в граммах
--
-- ===== ДОПОЛНИТЕЛЬНЫЕ ПОЛЯ =====
-- glycemic_index SMALLINT,        -- гликемический индекс
-- glycemic_load DECIMAL(5,2),     -- гликемическая нагрузка
