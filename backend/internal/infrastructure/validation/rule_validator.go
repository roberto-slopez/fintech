package validation

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/fintech-multipass/backend/internal/domain/entity"
	"github.com/fintech-multipass/backend/internal/infrastructure/database"
	"github.com/fintech-multipass/backend/internal/infrastructure/logger"
	"github.com/google/uuid"
)

// RuleValidator servicio para validación de reglas por país
type RuleValidator struct {
	db  *database.PostgresDB
	log *logger.Logger
}

// NewRuleValidator crea una nueva instancia del validador
func NewRuleValidator(db *database.PostgresDB, log *logger.Logger) *RuleValidator {
	return &RuleValidator{
		db:  db,
		log: log,
	}
}

// ValidateApplication valida una solicitud según las reglas del país
func (v *RuleValidator) ValidateApplication(ctx context.Context, app *entity.CreditApplication, rules []entity.CountryRule) ([]entity.ValidationResult, error) {
	var results []entity.ValidationResult

	for _, rule := range rules {
		if !rule.IsActive {
			continue
		}

		result := v.evaluateRule(app, rule)
		results = append(results, result)

		v.log.Debug().
			Str("rule", rule.Name).
			Bool("passed", result.Passed).
			Str("message", result.Message).
			Msg("Rule evaluated")
	}

	return results, nil
}

// evaluateRule evalúa una regla específica
func (v *RuleValidator) evaluateRule(app *entity.CreditApplication, rule entity.CountryRule) entity.ValidationResult {
	result := entity.ValidationResult{
		RuleID:   rule.ID,
		RuleName: rule.Name,
		Passed:   true,
	}

	switch rule.RuleType {
	case entity.RuleTypeDocumentValidation:
		result = v.validateDocument(app, rule)
	case entity.RuleTypeIncomeCheck:
		result = v.validateIncome(app, rule)
	case entity.RuleTypeDebtRatio:
		result = v.validateDebtRatio(app, rule)
	case entity.RuleTypeCreditScore:
		result = v.validateCreditScore(app, rule)
	case entity.RuleTypeAmountThreshold:
		result = v.validateAmountThreshold(app, rule)
	default:
		result.Message = "Rule type not implemented"
	}

	result.RuleID = rule.ID
	result.RuleName = rule.Name

	return result
}

// validateDocument valida el documento según la configuración
func (v *RuleValidator) validateDocument(app *entity.CreditApplication, rule entity.CountryRule) entity.ValidationResult {
	result := entity.ValidationResult{Passed: true}

	// Obtener el documento requerido de la configuración
	requiredDoc, _ := rule.Config["required_document"].(string)
	if requiredDoc != "" && app.DocumentType != requiredDoc {
		result.Passed = false
		result.Message = fmt.Sprintf("Document type must be %s, got %s", requiredDoc, app.DocumentType)
		return result
	}

	// Validar formato si hay regex
	if validateChecksum, ok := rule.Config["validate_checksum"].(bool); ok && validateChecksum {
		if !v.validateDocumentChecksum(app.DocumentType, app.DocumentNumber) {
			result.Passed = false
			result.Message = "Document validation failed"
			return result
		}
	}

	result.Message = "Document validated successfully"
	return result
}

// validateDocumentChecksum valida el checksum de documentos por tipo
func (v *RuleValidator) validateDocumentChecksum(docType, docNumber string) bool {
	docNumber = strings.ToUpper(strings.ReplaceAll(docNumber, " ", ""))

	switch docType {
	case "DNI":
		return v.validateSpanishDNI(docNumber)
	case "NIE":
		return v.validateSpanishNIE(docNumber)
	case "NIF":
		return v.validatePortugueseNIF(docNumber)
	case "CURP":
		return v.validateMexicanCURP(docNumber)
	case "CPF":
		return v.validateBrazilianCPF(docNumber)
	case "CC":
		return v.validateColombianCC(docNumber)
	case "CF":
		return v.validateItalianCF(docNumber)
	default:
		return len(docNumber) >= 5 // Validación básica
	}
}

// Validaciones específicas por país

func (v *RuleValidator) validateSpanishDNI(dni string) bool {
	// Formato: 8 dígitos + 1 letra
	if len(dni) != 9 {
		return false
	}
	match, _ := regexp.MatchString(`^[0-9]{8}[A-Z]$`, dni)
	if !match {
		return false
	}

	// Validar letra de control
	letters := "TRWAGMYFPDXBNJZSQVHLCKE"
	num := dni[:8]
	var numInt int
	fmt.Sscanf(num, "%d", &numInt)
	expectedLetter := letters[numInt%23]
	return dni[8] == byte(expectedLetter)
}

func (v *RuleValidator) validateSpanishNIE(nie string) bool {
	if len(nie) != 9 {
		return false
	}
	match, _ := regexp.MatchString(`^[XYZ][0-9]{7}[A-Z]$`, nie)
	return match
}

func (v *RuleValidator) validatePortugueseNIF(nif string) bool {
	if len(nif) != 9 {
		return false
	}
	match, _ := regexp.MatchString(`^[0-9]{9}$`, nif)
	return match
}

func (v *RuleValidator) validateMexicanCURP(curp string) bool {
	if len(curp) != 18 {
		return false
	}
	match, _ := regexp.MatchString(`^[A-Z]{4}[0-9]{6}[HM][A-Z]{5}[0-9A-Z][0-9]$`, curp)
	return match
}

func (v *RuleValidator) validateBrazilianCPF(cpf string) bool {
	// Remover caracteres no numéricos
	cpf = regexp.MustCompile(`[^0-9]`).ReplaceAllString(cpf, "")
	if len(cpf) != 11 {
		return false
	}

	// Verificar si todos los dígitos son iguales
	allEqual := true
	for i := 1; i < len(cpf); i++ {
		if cpf[i] != cpf[0] {
			allEqual = false
			break
		}
	}
	if allEqual {
		return false
	}

	// Validar dígitos verificadores
	return v.validateCPFDigits(cpf)
}

func (v *RuleValidator) validateCPFDigits(cpf string) bool {
	// Primer dígito verificador
	sum := 0
	for i := 0; i < 9; i++ {
		sum += int(cpf[i]-'0') * (10 - i)
	}
	remainder := sum % 11
	digit1 := 0
	if remainder >= 2 {
		digit1 = 11 - remainder
	}
	if int(cpf[9]-'0') != digit1 {
		return false
	}

	// Segundo dígito verificador
	sum = 0
	for i := 0; i < 10; i++ {
		sum += int(cpf[i]-'0') * (11 - i)
	}
	remainder = sum % 11
	digit2 := 0
	if remainder >= 2 {
		digit2 = 11 - remainder
	}
	return int(cpf[10]-'0') == digit2
}

func (v *RuleValidator) validateColombianCC(cc string) bool {
	// Cédula colombiana: 6-10 dígitos
	cc = regexp.MustCompile(`[^0-9]`).ReplaceAllString(cc, "")
	return len(cc) >= 6 && len(cc) <= 10
}

func (v *RuleValidator) validateItalianCF(cf string) bool {
	if len(cf) != 16 {
		return false
	}
	match, _ := regexp.MatchString(`^[A-Z]{6}[0-9]{2}[A-Z][0-9]{2}[A-Z][0-9]{3}[A-Z]$`, cf)
	return match
}

// validateIncome valida la relación ingreso/monto solicitado
func (v *RuleValidator) validateIncome(app *entity.CreditApplication, rule entity.CountryRule) entity.ValidationResult {
	result := entity.ValidationResult{Passed: true}

	maxMultiplier, ok := rule.Config["max_income_multiplier"].(float64)
	if !ok {
		maxMultiplier = 6.0 // Default
	}

	maxAllowed := app.MonthlyIncome * maxMultiplier
	if app.RequestedAmount > maxAllowed {
		result.Passed = false
		result.Message = fmt.Sprintf("Requested amount (%.2f) exceeds %.0fx monthly income (max: %.2f)",
			app.RequestedAmount, maxMultiplier, maxAllowed)
		return result
	}

	result.Message = fmt.Sprintf("Income check passed: %.2f <= %.2f", app.RequestedAmount, maxAllowed)
	return result
}

// validateDebtRatio valida la relación deuda/ingreso
func (v *RuleValidator) validateDebtRatio(app *entity.CreditApplication, rule entity.CountryRule) entity.ValidationResult {
	result := entity.ValidationResult{Passed: true}

	maxRatio, ok := rule.Config["max_ratio"].(float64)
	if !ok {
		maxRatio = 0.40 // Default 40%
	}

	// Si tenemos información bancaria, usarla
	var totalDebt float64
	if app.BankingInfo != nil && app.BankingInfo.TotalDebt != nil {
		totalDebt = *app.BankingInfo.TotalDebt
	}

	// Agregar el monto solicitado a la deuda
	totalDebt += app.RequestedAmount

	// Calcular ratio
	ratio := totalDebt / (app.MonthlyIncome * 12) // Deuda anual vs ingreso anual
	if ratio > maxRatio {
		result.Passed = false
		result.Message = fmt.Sprintf("Debt-to-income ratio (%.2f%%) exceeds maximum (%.2f%%)",
			ratio*100, maxRatio*100)
		return result
	}

	result.Message = fmt.Sprintf("Debt ratio check passed: %.2f%% <= %.2f%%", ratio*100, maxRatio*100)
	return result
}

// validateCreditScore valida el score crediticio mínimo
func (v *RuleValidator) validateCreditScore(app *entity.CreditApplication, rule entity.CountryRule) entity.ValidationResult {
	result := entity.ValidationResult{Passed: true}

	minScore, ok := rule.Config["min_score"].(float64)
	if !ok {
		minScore = 600 // Default
	}

	if app.BankingInfo == nil || app.BankingInfo.CreditScore == nil {
		result.Passed = false
		result.Message = "Credit score not available"
		result.RequiresReview = true
		return result
	}

	if float64(*app.BankingInfo.CreditScore) < minScore {
		result.Passed = false
		result.Message = fmt.Sprintf("Credit score (%d) below minimum (%.0f)",
			*app.BankingInfo.CreditScore, minScore)
		return result
	}

	result.Message = fmt.Sprintf("Credit score check passed: %d >= %.0f", *app.BankingInfo.CreditScore, minScore)
	return result
}

// validateAmountThreshold valida si el monto requiere revisión adicional
func (v *RuleValidator) validateAmountThreshold(app *entity.CreditApplication, rule entity.CountryRule) entity.ValidationResult {
	result := entity.ValidationResult{Passed: true}

	threshold, ok := rule.Config["threshold"].(float64)
	if !ok {
		threshold = 30000 // Default
	}

	action, _ := rule.Config["action"].(string)

	if app.RequestedAmount >= threshold {
		if action == "REQUIRE_REVIEW" {
			result.RequiresReview = true
			result.Message = fmt.Sprintf("Amount (%.2f) exceeds threshold (%.2f) - requires review",
				app.RequestedAmount, threshold)
		} else if action == "REJECT" {
			result.Passed = false
			result.Message = fmt.Sprintf("Amount (%.2f) exceeds maximum threshold (%.2f)",
				app.RequestedAmount, threshold)
		}
		return result
	}

	result.Message = "Amount within threshold"
	return result
}

// ValidateDocument valida un documento individual
func (v *RuleValidator) ValidateDocument(ctx context.Context, docType, docNumber, countryCode string) (bool, string, error) {
	// Obtener el regex de validación de la base de datos
	query := `
		SELECT dt.validation_regex 
		FROM document_types dt
		JOIN countries c ON dt.country_id = c.id
		WHERE c.code = $1 AND dt.code = $2
	`

	var validationRegex string
	row := v.db.QueryRow(ctx, query, countryCode, docType)
	if err := row.Scan(&validationRegex); err != nil {
		return false, "Document type not found for country", err
	}

	// Si hay regex, validar
	if validationRegex != "" {
		docNumber = strings.ToUpper(strings.ReplaceAll(docNumber, " ", ""))
		match, _ := regexp.MatchString(validationRegex, docNumber)
		if !match {
			return false, "Document format is invalid", nil
		}
	}

	// Validar checksum si aplica
	if !v.validateDocumentChecksum(docType, docNumber) {
		return false, "Document validation failed", nil
	}

	return true, "Document is valid", nil
}

// GetRulesForCountry obtiene las reglas activas de un país
func (v *RuleValidator) GetRulesForCountry(ctx context.Context, countryID uuid.UUID) ([]entity.CountryRule, error) {
	query := `
		SELECT id, country_id, rule_type, name, description, is_active, priority, config, created_at, updated_at
		FROM country_rules
		WHERE country_id = $1 AND is_active = true
		ORDER BY priority DESC
	`

	rows, err := v.db.Query(ctx, query, countryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []entity.CountryRule
	for rows.Next() {
		var rule entity.CountryRule
		var configJSON []byte
		if err := rows.Scan(
			&rule.ID, &rule.CountryID, &rule.RuleType, &rule.Name,
			&rule.Description, &rule.IsActive, &rule.Priority, &configJSON,
			&rule.CreatedAt, &rule.UpdatedAt,
		); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

