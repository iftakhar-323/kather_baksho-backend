package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"kather_baksho/database"
	"kather_baksho/models"
)

// PlantDiagnosisResult represents the structured response returned by the AI Plant Doctor.
type PlantDiagnosisResult struct {
	PlantName           string   `json:"plant_name"`
	Diagnosis           string   `json:"diagnosis"`
	Severity            string   `json:"severity"` // Mild, Moderate, Severe
	Confidence          float64  `json:"confidence"`
	Cause               string   `json:"cause"`
	ActionPlan          []string `json:"action_plan"`
	RecommendedProducts []string `json:"recommended_products"`
	ProviderUsed        string   `json:"provider_used"` // "gemini-1.5-flash" or "local-botanical-expert-fallback"
}

// DiagnosePlant analyzes symptoms using Gemini API when available, falling back seamlessly to the local expert engine.
func DiagnosePlant(plantName, symptoms, environment string) PlantDiagnosisResult {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey != "" {
		result, err := callGeminiDiagnosis(apiKey, plantName, symptoms, environment)
		if err == nil {
			result.ProviderUsed = "gemini-1.5-flash"
			return result
		}
		// If Gemini fails, proceed automatically to local fallback
	}

	return localBotanicalExpertDiagnosis(plantName, symptoms, environment)
}

// AnswerPlantCareQuestion provides conversational answers with fallback support.
func AnswerPlantCareQuestion(userQuestion, plantContext string) string {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey != "" {
		ans, err := callGeminiChat(apiKey, userQuestion, plantContext)
		if err == nil {
			return ans
		}
	}
	return localBotanicalExpertChat(userQuestion)
}

// ---------- Gemini API Integration ----------

func callGeminiDiagnosis(apiKey, plantName, symptoms, environment string) (PlantDiagnosisResult, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=%s", apiKey)

	prompt := fmt.Sprintf(`You are an expert botanical pathologist and plant doctor. Analyze the following plant symptoms and provide a diagnosis in strict JSON format with keys:
"plant_name": string,
"diagnosis": string,
"severity": "Mild"|"Moderate"|"Severe",
"confidence": float (0.0 to 1.0),
"cause": string,
"action_plan": list of strings,
"recommended_products": list of strings.

Plant: %s
Symptoms: %s
Environment: %s

Respond ONLY with valid JSON.`, plantName, symptoms, environment)

	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{{"text": prompt}},
			},
		},
	}

	bodyBytes, _ := json.Marshal(reqBody)
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return PlantDiagnosisResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return PlantDiagnosisResult{}, fmt.Errorf("gemini api returned status %d", resp.StatusCode)
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return PlantDiagnosisResult{}, err
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(respBytes, &geminiResp); err != nil || len(geminiResp.Candidates) == 0 {
		return PlantDiagnosisResult{}, fmt.Errorf("failed to parse gemini response")
	}

	text := geminiResp.Candidates[0].Content.Parts[0].Text
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	var result PlantDiagnosisResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return PlantDiagnosisResult{}, err
	}

	return result, nil
}

func callGeminiChat(apiKey, question, context string) (string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=%s", apiKey)
	prompt := fmt.Sprintf("You are KatherBox's friendly AI Nursery Assistant. Answer this plant question concisely in warm English/Bengali context: %s (Plant: %s)", question, context)

	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{{"text": prompt}},
			},
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(respBytes, &geminiResp); err != nil || len(geminiResp.Candidates) == 0 {
		return "", fmt.Errorf("failed to parse chat response")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

// ---------- Local Botanical Expert Fallback Engine ----------

func localBotanicalExpertDiagnosis(plantName, symptoms, environment string) PlantDiagnosisResult {
	if plantName == "" {
		plantName = "Houseplant"
	}
	symLower := strings.ToLower(symptoms + " " + environment)

	var diagnosis, severity, cause string
	var confidence float64
	var plan []string
	var products []string

	switch {
	case strings.Contains(symLower, "yellow") || strings.Contains(symLower, "হলুদ"):
		diagnosis = "Overwatering / Chlorosis (পানির আধিক্য বা নাইট্রোজেনের ঘাটতি)"
		severity = "Moderate"
		confidence = 0.92
		cause = "Soil moisture retention is excessive, suffocating roots and hindering chlorophyll synthesis."
		plan = []string{
			"Allow the top 2 inches of soil to completely dry out between watering.",
			"Check that the pot has proper drainage holes to avoid standing water.",
			"Apply a balanced organic nitrogen-rich fertilizer once every 2 weeks.",
		}
		products = []string{"Organic Plant Booster Fertilizer", "Terracotta Pot with Drainage Tray", "Moisture Meter Sensor"}

	case strings.Contains(symLower, "brown") || strings.Contains(symLower, "বাদামী") || strings.Contains(symLower, "dry tips") || strings.Contains(symLower, "crispy"):
		diagnosis = "Low Humidity / Underwatering (কম আর্দ্রতা বা পানির অভাব)"
		severity = "Mild"
		confidence = 0.89
		cause = "Dry ambient air and insufficient deep watering causing transpiration stress at leaf margins."
		plan = []string{
			"Water deeply until water drains freely from the pot base.",
			"Mist the leaves twice weekly or place a pebble tray with water beneath the pot.",
			"Keep the plant away from direct air conditioner or heating drafts.",
		}
		products = []string{"Continuous Fine Mist Spray Bottle", "Sphagnum Moss Humidifier Ring", "Self-Watering Planter"}

	case strings.Contains(symLower, "pest") || strings.Contains(symLower, "bug") || strings.Contains(symLower, "পোকা") || strings.Contains(symLower, "mite") || strings.Contains(symLower, "white"):
		diagnosis = "Foliar Pest Infestation (Mealybugs or Spider Mites / মিলিবাগ বা লাল মাকড়)"
		severity = "Severe"
		confidence = 0.94
		cause = "Insect pests feeding on plant sap and excreting honeydew on foliage."
		plan = []string{
			"Isolate the plant immediately to prevent spreading to other plants.",
			"Wipe the affected leaves and stems with dilute neem oil solution.",
			"Spray thoroughly every 5 days for 3 cycles until completely clear.",
		}
		products = []string{"Cold-Pressed Organic Neem Oil Spray", "Insecticidal Soap Wash", "Microfiber Leaf Cleaning Gloves"}

	case strings.Contains(symLower, "droop") || strings.Contains(symLower, "wilt") || strings.Contains(symLower, "ঝুলে"):
		diagnosis = "Transplantation Shock / Root Hypoxia (রুট শক বা অক্সিজেনের ঘাটতি)"
		severity = "Moderate"
		confidence = 0.86
		cause = "Disturbed root system or sudden thermal/environmental transition."
		plan = []string{
			"Keep in bright indirect sunlight; avoid direct harsh sun for 7 days.",
			"Do not add chemical fertilizer while the roots are stressed.",
			"Maintain evenly moist, well-aerated potting mix.",
		}
		products = []string{"Perlite & Coco Peat Aerated Potting Mix", "Rooting Hormone & Tonic", "Indoor Grow Light"}

	default:
		diagnosis = "General Nutrient Imbalance & Environmental Stress"
		severity = "Mild"
		confidence = 0.78
		cause = "Inconsistent light exposure, soil compaction, or micronutrient depletion."
		plan = []string{
			"Rotate the plant weekly for even light absorption.",
			"Gently aerate topsoil with a hand fork to improve oxygen permeability.",
			"Feed with diluted seaweed or vermicompost fertilizer.",
		}
		products = []string{"Organic Vermicompost", "Mini Gardening Tool Set", "All-Purpose Liquid Plant Food"}
	}

	// Match available product IDs from database if present and DB is connected
	var matchedProducts []string
	for _, prodName := range products {
		if database.DB != nil {
			var p models.Product
			if err := database.DB.Where("name LIKE ?", "%"+prodName+"%").First(&p).Error; err == nil {
				matchedProducts = append(matchedProducts, p.Name)
				continue
			}
		}
		matchedProducts = append(matchedProducts, prodName)
	}

	return PlantDiagnosisResult{
		PlantName:           plantName,
		Diagnosis:           diagnosis,
		Severity:            severity,
		Confidence:          confidence,
		Cause:               cause,
		ActionPlan:          plan,
		RecommendedProducts: matchedProducts,
		ProviderUsed:        "local-botanical-expert-fallback",
	}
}

func localBotanicalExpertChat(question string) string {
	q := strings.ToLower(question)
	switch {
	case strings.Contains(q, "water") || strings.Contains(q, "পানি"):
		return "Most indoor plants thrive when watered once the top 1-2 inches of soil feel dry. Always ensure pots have drainage holes to prevent root rot."
	case strings.Contains(q, "sun") || strings.Contains(q, "আলো") || strings.Contains(q, "light"):
		return "Bright, indirect sunlight is ideal for most tropical houseplants (Monstera, Pothos, Snake Plant). Avoid harsh midday direct sunlight which scorches leaves."
	case strings.Contains(q, "fertilizer") || strings.Contains(q, "সার"):
		return "Feed your plants during their active growth phase (Spring & Summer) once every 2-4 weeks with organic fertilizer or vermicompost."
	default:
		return "To keep your plants thriving: provide bright indirect light, water only when soil dries slightly, and mist leaves occasionally for humidity. Ask me about watering, sunlight, or diagnosing yellow leaves!"
	}
}
