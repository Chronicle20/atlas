package context

import (
	"fmt"
	"strconv"
	"strings"
)

// EvaluateArithmeticExpression evaluates simple arithmetic expressions
// Supports: +, -, *, / operators
// Example: "10 * 5" -> 50, "100 / 2" -> 50
func EvaluateArithmeticExpression(expr string) (int, error) {
	expr = strings.TrimSpace(expr)

	// Check for multiplication
	if strings.Contains(expr, "*") {
		parts := strings.Split(expr, "*")
		if len(parts) != 2 {
			return 0, fmt.Errorf("invalid multiplication expression: %s", expr)
		}
		left, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return 0, fmt.Errorf("invalid left operand: %s", parts[0])
		}
		right, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return 0, fmt.Errorf("invalid right operand: %s", parts[1])
		}
		return left * right, nil
	}

	// Check for division
	if strings.Contains(expr, "/") {
		parts := strings.Split(expr, "/")
		if len(parts) != 2 {
			return 0, fmt.Errorf("invalid division expression: %s", expr)
		}
		left, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return 0, fmt.Errorf("invalid left operand: %s", parts[0])
		}
		right, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return 0, fmt.Errorf("invalid right operand: %s", parts[1])
		}
		if right == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return left / right, nil
	}

	// Check for addition
	if strings.Contains(expr, "+") {
		parts := strings.Split(expr, "+")
		if len(parts) != 2 {
			return 0, fmt.Errorf("invalid addition expression: %s", expr)
		}
		left, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return 0, fmt.Errorf("invalid left operand: %s", parts[0])
		}
		right, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return 0, fmt.Errorf("invalid right operand: %s", parts[1])
		}
		return left + right, nil
	}

	// Check for subtraction (but not negative numbers)
	if strings.Contains(expr, "-") && !strings.HasPrefix(expr, "-") {
		parts := strings.Split(expr, "-")
		if len(parts) != 2 {
			return 0, fmt.Errorf("invalid subtraction expression: %s", expr)
		}
		left, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return 0, fmt.Errorf("invalid left operand: %s", parts[0])
		}
		right, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return 0, fmt.Errorf("invalid right operand: %s", parts[1])
		}
		return left - right, nil
	}

	// No operator found, try to parse as integer
	return strconv.Atoi(expr)
}

// EvaluateValueAsInt evaluates a string value as an integer, supporting arithmetic expressions
func EvaluateValueAsInt(value string) (int, error) {
	// Check if the value contains arithmetic operators
	if strings.ContainsAny(value, "+-*/") {
		// Evaluate arithmetic expression using the shared function
		return EvaluateArithmeticExpression(value)
	}

	// No arithmetic, convert directly to integer
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("value [%s] is not a valid integer", value)
	}

	return intValue, nil
}
