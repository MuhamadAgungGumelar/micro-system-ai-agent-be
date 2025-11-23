package workflow

import (
	"fmt"
	"reflect"
	"strings"
)

// ConditionEvaluator evaluates workflow conditions
type ConditionEvaluator struct{}

// NewConditionEvaluator creates a new condition evaluator
func NewConditionEvaluator() *ConditionEvaluator {
	return &ConditionEvaluator{}
}

// Evaluate evaluates all conditions against the provided data
// Returns true if all conditions pass (or if no conditions exist)
func (e *ConditionEvaluator) Evaluate(conditions []Condition, data map[string]interface{}) (bool, error) {
	if len(conditions) == 0 {
		return true, nil // No conditions means always pass
	}

	// Check if any condition specifies OR logic
	hasOR := false
	for _, condition := range conditions {
		if strings.ToUpper(condition.Logic) == "OR" {
			hasOR = true
			break
		}
	}

	if hasOR {
		// OR logic: at least one condition must pass
		for _, condition := range conditions {
			result, err := e.evaluateSingle(condition, data)
			if err != nil {
				return false, err
			}
			if result {
				return true, nil // One condition passed, return true
			}
		}
		return false, nil // No conditions passed
	} else {
		// AND logic: all conditions must pass
		for _, condition := range conditions {
			result, err := e.evaluateSingle(condition, data)
			if err != nil {
				return false, err
			}
			if !result {
				return false, nil // One condition failed, return false
			}
		}
		return true, nil // All conditions passed
	}
}

// evaluateSingle evaluates a single condition
func (e *ConditionEvaluator) evaluateSingle(condition Condition, data map[string]interface{}) (bool, error) {
	// Extract field value from data
	fieldValue, exists := data[condition.Field]
	if !exists {
		// Field doesn't exist in data
		// For "not_equals", this should return true
		if condition.Operator == "not_equals" {
			return true, nil
		}
		return false, fmt.Errorf("field '%s' not found in data", condition.Field)
	}

	// Evaluate based on operator
	switch condition.Operator {
	case "equals":
		return e.compareEquals(fieldValue, condition.Value), nil

	case "not_equals":
		return !e.compareEquals(fieldValue, condition.Value), nil

	case "greater_than":
		return e.compareGreaterThan(fieldValue, condition.Value)

	case "greater_or_equal":
		return e.compareGreaterOrEqual(fieldValue, condition.Value)

	case "less_than":
		return e.compareLessThan(fieldValue, condition.Value)

	case "less_or_equal":
		return e.compareLessOrEqual(fieldValue, condition.Value)

	case "contains":
		result, err := e.compareContains(fieldValue, condition.Value)
		return result, err

	case "not_contains":
		result, err := e.compareContains(fieldValue, condition.Value)
		if err != nil {
			return false, err
		}
		return !result, nil

	case "starts_with":
		return e.compareStartsWith(fieldValue, condition.Value)

	case "ends_with":
		return e.compareEndsWith(fieldValue, condition.Value)

	case "in_list":
		result, err := e.compareInList(fieldValue, condition.Value)
		return result, err

	case "not_in_list":
		result, err := e.compareInList(fieldValue, condition.Value)
		if err != nil {
			return false, err
		}
		return !result, nil

	default:
		return false, fmt.Errorf("unknown operator: %s", condition.Operator)
	}
}

// compareEquals checks if two values are equal
func (e *ConditionEvaluator) compareEquals(fieldValue, conditionValue interface{}) bool {
	return reflect.DeepEqual(fieldValue, conditionValue)
}

// compareGreaterThan checks if field value is greater than condition value
func (e *ConditionEvaluator) compareGreaterThan(fieldValue, conditionValue interface{}) (bool, error) {
	fieldNum, err := toFloat64(fieldValue)
	if err != nil {
		return false, fmt.Errorf("field value is not a number: %v", err)
	}

	condNum, err := toFloat64(conditionValue)
	if err != nil {
		return false, fmt.Errorf("condition value is not a number: %v", err)
	}

	return fieldNum > condNum, nil
}

// compareGreaterOrEqual checks if field value is greater than or equal to condition value
func (e *ConditionEvaluator) compareGreaterOrEqual(fieldValue, conditionValue interface{}) (bool, error) {
	fieldNum, err := toFloat64(fieldValue)
	if err != nil {
		return false, fmt.Errorf("field value is not a number: %v", err)
	}

	condNum, err := toFloat64(conditionValue)
	if err != nil {
		return false, fmt.Errorf("condition value is not a number: %v", err)
	}

	return fieldNum >= condNum, nil
}

// compareLessThan checks if field value is less than condition value
func (e *ConditionEvaluator) compareLessThan(fieldValue, conditionValue interface{}) (bool, error) {
	fieldNum, err := toFloat64(fieldValue)
	if err != nil {
		return false, fmt.Errorf("field value is not a number: %v", err)
	}

	condNum, err := toFloat64(conditionValue)
	if err != nil {
		return false, fmt.Errorf("condition value is not a number: %v", err)
	}

	return fieldNum < condNum, nil
}

// compareLessOrEqual checks if field value is less than or equal to condition value
func (e *ConditionEvaluator) compareLessOrEqual(fieldValue, conditionValue interface{}) (bool, error) {
	fieldNum, err := toFloat64(fieldValue)
	if err != nil {
		return false, fmt.Errorf("field value is not a number: %v", err)
	}

	condNum, err := toFloat64(conditionValue)
	if err != nil {
		return false, fmt.Errorf("condition value is not a number: %v", err)
	}

	return fieldNum <= condNum, nil
}

// compareContains checks if field value contains condition value (string)
func (e *ConditionEvaluator) compareContains(fieldValue, conditionValue interface{}) (bool, error) {
	fieldStr, ok := fieldValue.(string)
	if !ok {
		return false, fmt.Errorf("field value is not a string")
	}

	condStr, ok := conditionValue.(string)
	if !ok {
		return false, fmt.Errorf("condition value is not a string")
	}

	return strings.Contains(strings.ToLower(fieldStr), strings.ToLower(condStr)), nil
}

// compareStartsWith checks if field value starts with condition value
func (e *ConditionEvaluator) compareStartsWith(fieldValue, conditionValue interface{}) (bool, error) {
	fieldStr, ok := fieldValue.(string)
	if !ok {
		return false, fmt.Errorf("field value is not a string")
	}

	condStr, ok := conditionValue.(string)
	if !ok {
		return false, fmt.Errorf("condition value is not a string")
	}

	return strings.HasPrefix(strings.ToLower(fieldStr), strings.ToLower(condStr)), nil
}

// compareEndsWith checks if field value ends with condition value
func (e *ConditionEvaluator) compareEndsWith(fieldValue, conditionValue interface{}) (bool, error) {
	fieldStr, ok := fieldValue.(string)
	if !ok {
		return false, fmt.Errorf("field value is not a string")
	}

	condStr, ok := conditionValue.(string)
	if !ok {
		return false, fmt.Errorf("condition value is not a string")
	}

	return strings.HasSuffix(strings.ToLower(fieldStr), strings.ToLower(condStr)), nil
}

// compareInList checks if field value is in the condition list
func (e *ConditionEvaluator) compareInList(fieldValue, conditionValue interface{}) (bool, error) {
	condList, ok := conditionValue.([]interface{})
	if !ok {
		return false, fmt.Errorf("condition value is not a list")
	}

	for _, item := range condList {
		if reflect.DeepEqual(fieldValue, item) {
			return true, nil
		}
	}

	return false, nil
}

// toFloat64 converts various numeric types to float64
func toFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case int32:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", value)
	}
}
