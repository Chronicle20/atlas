'use client';

import React, { useState } from 'react';
import { AlertCircle, CheckCircle, Loader2, XCircle } from 'lucide-react';
import type { Conversation } from '@/types/models/conversation';
import { Button } from '@/components/ui/button';

interface ValidationError {
  stateId: string;
  field: string;
  errorType: string;
  message: string;
}

interface ValidationResult {
  valid: boolean;
  errors: ValidationError[];
}

interface ValidationFeedbackProps {
  conversation: Conversation | null;
  apiBaseUrl: string;
}

export function ValidationFeedback({ conversation, apiBaseUrl }: ValidationFeedbackProps) {
  const [isValidating, setIsValidating] = useState(false);
  const [validationResult, setValidationResult] = useState<ValidationResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleValidate = async () => {
    if (!conversation) {
      setError('No conversation to validate');
      return;
    }

    setIsValidating(true);
    setError(null);
    setValidationResult(null);

    try {
      const response = await fetch(`${apiBaseUrl}/npcs/conversations/validate`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/vnd.api+json',
        },
        body: JSON.stringify({
          data: conversation
        })
      });

      if (!response.ok) {
        throw new Error(`Validation request failed: ${response.statusText}`);
      }

      const result = await response.json();
      setValidationResult(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred during validation');
    } finally {
      setIsValidating(false);
    }
  };

  // Group errors by state
  const errorsByState = React.useMemo(() => {
    if (!validationResult || validationResult.valid) return new Map<string, ValidationError[]>();

    const grouped = new Map<string, ValidationError[]>();
    for (const error of validationResult.errors) {
      const stateId = error.stateId || '(general)';
      if (!grouped.has(stateId)) {
        grouped.set(stateId, []);
      }
      grouped.get(stateId)!.push(error);
    }
    return grouped;
  }, [validationResult]);

  // Get error type display info
  const getErrorTypeInfo = (errorType: string) => {
    switch (errorType) {
      case 'required':
        return { icon: XCircle, color: 'text-red-600', bgColor: 'bg-red-50', borderColor: 'border-red-200' };
      case 'invalid':
      case 'invalid_count':
      case 'invalid_reference':
        return { icon: AlertCircle, color: 'text-orange-600', bgColor: 'bg-orange-50', borderColor: 'border-orange-200' };
      case 'duplicate':
        return { icon: AlertCircle, color: 'text-yellow-600', bgColor: 'bg-yellow-50', borderColor: 'border-yellow-200' };
      case 'unreachable':
      case 'circular_reference':
      case 'infinite_loop':
        return { icon: AlertCircle, color: 'text-purple-600', bgColor: 'bg-purple-50', borderColor: 'border-purple-200' };
      default:
        return { icon: XCircle, color: 'text-gray-600', bgColor: 'bg-gray-50', borderColor: 'border-gray-200' };
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold">Validation</h3>
        <Button
          onClick={handleValidate}
          disabled={!conversation || isValidating}
          variant="outline"
          size="sm"
        >
          {isValidating ? (
            <>
              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              Validating...
            </>
          ) : (
            'Validate Conversation'
          )}
        </Button>
      </div>

      {error && (
        <div className="border border-red-300 rounded p-3 bg-red-50">
          <div className="flex items-start gap-2">
            <XCircle className="h-5 w-5 text-red-600 flex-shrink-0 mt-0.5" />
            <div>
              <h4 className="text-sm font-semibold text-red-800">Validation Error</h4>
              <p className="text-sm text-red-700 mt-1">{error}</p>
            </div>
          </div>
        </div>
      )}

      {validationResult && (
        <>
          {validationResult.valid ? (
            <div className="border border-green-300 rounded p-4 bg-green-50">
              <div className="flex items-start gap-3">
                <CheckCircle className="h-6 w-6 text-green-600 flex-shrink-0 mt-0.5" />
                <div>
                  <h4 className="text-base font-semibold text-green-800">Validation Passed</h4>
                  <p className="text-sm text-green-700 mt-1">
                    The conversation has no validation errors and is ready to use.
                  </p>
                </div>
              </div>
            </div>
          ) : (
            <div className="space-y-3">
              <div className="border border-red-300 rounded p-3 bg-red-50">
                <div className="flex items-start gap-2">
                  <XCircle className="h-5 w-5 text-red-600 flex-shrink-0 mt-0.5" />
                  <div>
                    <h4 className="text-sm font-semibold text-red-800">
                      Validation Failed ({validationResult.errors.length} error{validationResult.errors.length !== 1 ? 's' : ''})
                    </h4>
                    <p className="text-xs text-red-700 mt-1">
                      Please fix the following errors before saving this conversation.
                    </p>
                  </div>
                </div>
              </div>

              <div className="space-y-3">
                {Array.from(errorsByState.entries()).map(([stateId, errors]) => (
                  <div key={stateId} className="border border-gray-300 rounded p-3 bg-white">
                    <h5 className="text-sm font-semibold text-gray-800 mb-2">
                      {stateId === '(general)' ? 'General Errors' : `State: ${stateId}`}
                    </h5>
                    <div className="space-y-2">
                      {errors.map((error, index) => {
                        const typeInfo = getErrorTypeInfo(error.errorType);
                        const Icon = typeInfo.icon;

                        return (
                          <div
                            key={index}
                            className={`border ${typeInfo.borderColor} rounded p-2 ${typeInfo.bgColor}`}
                          >
                            <div className="flex items-start gap-2">
                              <Icon className={`h-4 w-4 ${typeInfo.color} flex-shrink-0 mt-0.5`} />
                              <div className="flex-1">
                                <div className="flex items-start justify-between gap-2">
                                  <div>
                                    <p className="text-sm font-medium text-gray-900">{error.message}</p>
                                    <div className="mt-1 flex gap-3 text-xs text-gray-600">
                                      <span>
                                        <span className="font-medium">Field:</span> {error.field || 'N/A'}
                                      </span>
                                      <span>
                                        <span className="font-medium">Type:</span> {error.errorType}
                                      </span>
                                    </div>
                                  </div>
                                </div>
                              </div>
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </>
      )}

      {!validationResult && !error && !isValidating && (
        <div className="text-sm text-gray-600 bg-blue-50 p-3 rounded border border-blue-200">
          <p className="font-semibold mb-1">About Validation:</p>
          <ul className="list-disc list-inside space-y-1">
            <li>Click "Validate Conversation" to check for errors</li>
            <li>Validation checks for missing required fields, invalid references, and structural issues</li>
            <li>Fix all errors before saving to ensure the conversation works correctly</li>
            <li>Validation is performed server-side for accuracy</li>
          </ul>
        </div>
      )}
    </div>
  );
}
