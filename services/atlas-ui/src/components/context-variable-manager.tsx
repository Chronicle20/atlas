'use client';

import React, { useMemo } from 'react';
import { AlertCircle, CheckCircle, Info } from 'lucide-react';
import type { Conversation } from '@/types/models/conversation';

interface ContextVariableManagerProps {
  conversation: Conversation | null;
}

interface VariableInfo {
  name: string;
  setIn: string[];
  usedIn: string[];
  isSet: boolean;
  isUsed: boolean;
}

// Extract context variable references from a string
function extractContextVars(str: string): string[] {
  if (!str) return [];

  const vars: string[] = [];

  // Match {context.variable} format
  const bracketMatches = str.matchAll(/\{context\.([a-zA-Z_][a-zA-Z0-9_]*)\}/g);
  for (const match of bracketMatches) {
    vars.push(match[1]!);
  }

  // Match context.variable format (not in braces)
  const directMatches = str.matchAll(/(?<![{a-zA-Z0-9_])context\.([a-zA-Z_][a-zA-Z0-9_]*)/g);
  for (const match of directMatches) {
    // Avoid duplicates
    if (!vars.includes(match[1]!)) {
      vars.push(match[1]!);
    }
  }

  return vars;
}

export function ContextVariableManager({ conversation }: ContextVariableManagerProps) {
  const variableInfo = useMemo<Map<string, VariableInfo>>(() => {
    const infoMap = new Map<string, VariableInfo>();

    if (!conversation) return infoMap;

    const getOrCreate = (name: string): VariableInfo => {
      if (!infoMap.has(name)) {
        infoMap.set(name, {
          name,
          setIn: [],
          usedIn: [],
          isSet: false,
          isUsed: false
        });
      }
      return infoMap.get(name)!;
    };

    // Scan all states
    for (const state of conversation.attributes.states) {
      const stateId = state.id;

      // Check askNumber states (they set context variables)
      if (state.type === 'askNumber' && state.askNumber?.contextKey) {
        const info = getOrCreate(state.askNumber.contextKey);
        info.isSet = true;
        info.setIn.push(`${stateId} (askNumber)`);
      }

      // Check askStyle states (they set context variables)
      if (state.type === 'askStyle' && state.askStyle?.contextKey) {
        const info = getOrCreate(state.askStyle.contextKey);
        info.isSet = true;
        info.setIn.push(`${stateId} (askStyle)`);
      }

      // Check dialogue choices (they can set context variables)
      if (state.type === 'dialogue' && state.dialogue?.choices) {
        for (let i = 0; i < state.dialogue.choices.length; i++) {
          const choice = state.dialogue.choices[i]!;
          if (choice.context) {
            Object.keys(choice.context).forEach(key => {
              const info = getOrCreate(key);
              info.isSet = true;
              info.setIn.push(`${stateId} (dialogue choice ${i + 1})`);
            });
          }
        }
      }

      // Check genericAction operations for variable usage
      if (state.type === 'genericAction' && state.genericAction) {
        // Check operations
        state.genericAction.operations?.forEach((op, opIndex) => {
          Object.entries(op.params || {}).forEach(([paramName, paramValue]) => {
            const vars = extractContextVars(String(paramValue));
            vars.forEach(varName => {
              const info = getOrCreate(varName);
              info.isUsed = true;
              info.usedIn.push(`${stateId} (operation ${opIndex + 1} - ${paramName})`);
            });
          });
        });

        // Check conditions
        state.genericAction.outcomes?.forEach((outcome, outcomeIndex) => {
          outcome.conditions?.forEach((cond, condIndex) => {
            // Check value
            const valueVars = extractContextVars(cond.value);
            valueVars.forEach(varName => {
              const info = getOrCreate(varName);
              info.isUsed = true;
              info.usedIn.push(`${stateId} (outcome ${outcomeIndex + 1}, condition ${condIndex + 1} - value)`);
            });

            // Check referenceId
            if (cond.referenceId) {
              const refVars = extractContextVars(cond.referenceId);
              refVars.forEach(varName => {
                const info = getOrCreate(varName);
                info.isUsed = true;
                info.usedIn.push(`${stateId} (outcome ${outcomeIndex + 1}, condition ${condIndex + 1} - referenceId)`);
              });
            }

            // Check worldId
            if (cond.worldId) {
              const worldVars = extractContextVars(String(cond.worldId));
              worldVars.forEach(varName => {
                const info = getOrCreate(varName);
                info.isUsed = true;
                info.usedIn.push(`${stateId} (outcome ${outcomeIndex + 1}, condition ${condIndex + 1} - worldId)`);
              });
            }

            // Check channelId
            if (cond.channelId) {
              const channelVars = extractContextVars(String(cond.channelId));
              channelVars.forEach(varName => {
                const info = getOrCreate(varName);
                info.isUsed = true;
                info.usedIn.push(`${stateId} (outcome ${outcomeIndex + 1}, condition ${condIndex + 1} - channelId)`);
              });
            }
          });
        });
      }

      // Check askNumber text for variable usage
      if (state.type === 'askNumber' && state.askNumber?.text) {
        const vars = extractContextVars(state.askNumber.text);
        vars.forEach(varName => {
          const info = getOrCreate(varName);
          info.isUsed = true;
          info.usedIn.push(`${stateId} (askNumber text)`);
        });
      }

      // Check askStyle for variable usage
      if (state.type === 'askStyle' && state.askStyle) {
        // Check text
        if (state.askStyle.text) {
          const vars = extractContextVars(state.askStyle.text);
          vars.forEach(varName => {
            const info = getOrCreate(varName);
            info.isUsed = true;
            info.usedIn.push(`${stateId} (askStyle text)`);
          });
        }

        // Check stylesContextKey (this is a usage, not a set)
        if (state.askStyle.stylesContextKey) {
          const info = getOrCreate(state.askStyle.stylesContextKey);
          info.isUsed = true;
          info.usedIn.push(`${stateId} (askStyle stylesContextKey)`);
        }
      }

      // Check dialogue text for variable usage
      if (state.type === 'dialogue' && state.dialogue?.text) {
        const vars = extractContextVars(state.dialogue.text);
        vars.forEach(varName => {
          const info = getOrCreate(varName);
          info.isUsed = true;
          info.usedIn.push(`${stateId} (dialogue text)`);
        });
      }
    }

    return infoMap;
  }, [conversation]);

  const variables = useMemo(() => Array.from(variableInfo.values()), [variableInfo]);

  // Categorize variables
  const usedNotSet = variables.filter(v => v.isUsed && !v.isSet);
  const setNotUsed = variables.filter(v => v.isSet && !v.isUsed);
  const proper = variables.filter(v => v.isSet && v.isUsed);

  if (!conversation || variables.length === 0) {
    return (
      <div className="p-4 border border-gray-300 rounded bg-gray-50">
        <div className="flex items-center gap-2 text-gray-600">
          <Info className="h-5 w-5" />
          <p className="text-sm">No context variables detected in this conversation.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold">Context Variables</h3>
        <div className="text-sm text-gray-600">
          {variables.length} variable{variables.length !== 1 ? 's' : ''} found
        </div>
      </div>

      {/* Errors: Used but not set */}
      {usedNotSet.length > 0 && (
        <div className="border border-red-300 rounded p-3 bg-red-50">
          <div className="flex items-start gap-2 mb-2">
            <AlertCircle className="h-5 w-5 text-red-600 flex-shrink-0 mt-0.5" />
            <div className="flex-1">
              <h4 className="text-sm font-semibold text-red-800">
                Used but not set ({usedNotSet.length})
              </h4>
              <p className="text-xs text-red-700 mt-1">
                These variables are referenced but never assigned a value. This may cause runtime errors.
              </p>
            </div>
          </div>
          <div className="space-y-2 mt-3">
            {usedNotSet.map((variable) => (
              <div key={variable.name} className="bg-white border border-red-200 rounded p-2">
                <div className="font-mono text-sm font-semibold text-red-800">
                  {variable.name}
                </div>
                <div className="mt-1 text-xs text-gray-700">
                  <span className="font-medium">Used in:</span>
                  <ul className="list-disc list-inside ml-2 mt-1">
                    {variable.usedIn.map((location, idx) => (
                      <li key={idx}>{location}</li>
                    ))}
                  </ul>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Warnings: Set but not used */}
      {setNotUsed.length > 0 && (
        <div className="border border-yellow-300 rounded p-3 bg-yellow-50">
          <div className="flex items-start gap-2 mb-2">
            <AlertCircle className="h-5 w-5 text-yellow-600 flex-shrink-0 mt-0.5" />
            <div className="flex-1">
              <h4 className="text-sm font-semibold text-yellow-800">
                Set but not used ({setNotUsed.length})
              </h4>
              <p className="text-xs text-yellow-700 mt-1">
                These variables are assigned but never referenced. Consider removing them.
              </p>
            </div>
          </div>
          <div className="space-y-2 mt-3">
            {setNotUsed.map((variable) => (
              <div key={variable.name} className="bg-white border border-yellow-200 rounded p-2">
                <div className="font-mono text-sm font-semibold text-yellow-800">
                  {variable.name}
                </div>
                <div className="mt-1 text-xs text-gray-700">
                  <span className="font-medium">Set in:</span>
                  <ul className="list-disc list-inside ml-2 mt-1">
                    {variable.setIn.map((location, idx) => (
                      <li key={idx}>{location}</li>
                    ))}
                  </ul>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Success: Properly used */}
      {proper.length > 0 && (
        <div className="border border-green-300 rounded p-3 bg-green-50">
          <div className="flex items-start gap-2 mb-2">
            <CheckCircle className="h-5 w-5 text-green-600 flex-shrink-0 mt-0.5" />
            <div className="flex-1">
              <h4 className="text-sm font-semibold text-green-800">
                Properly used ({proper.length})
              </h4>
              <p className="text-xs text-green-700 mt-1">
                These variables are both set and used correctly.
              </p>
            </div>
          </div>
          <div className="space-y-2 mt-3">
            {proper.map((variable) => (
              <div key={variable.name} className="bg-white border border-green-200 rounded p-2">
                <div className="font-mono text-sm font-semibold text-green-800">
                  {variable.name}
                </div>
                <div className="grid grid-cols-2 gap-4 mt-1 text-xs text-gray-700">
                  <div>
                    <span className="font-medium">Set in:</span>
                    <ul className="list-disc list-inside ml-2 mt-1">
                      {variable.setIn.map((location, idx) => (
                        <li key={idx}>{location}</li>
                      ))}
                    </ul>
                  </div>
                  <div>
                    <span className="font-medium">Used in:</span>
                    <ul className="list-disc list-inside ml-2 mt-1">
                      {variable.usedIn.slice(0, 3).map((location, idx) => (
                        <li key={idx}>{location}</li>
                      ))}
                      {variable.usedIn.length > 3 && (
                        <li className="text-gray-500">...and {variable.usedIn.length - 3} more</li>
                      )}
                    </ul>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="text-xs text-gray-600 bg-blue-50 p-3 rounded border border-blue-200">
        <p className="font-semibold mb-1">Context Variable Usage:</p>
        <ul className="list-disc list-inside space-y-1">
          <li>Variables are set by askNumber, askStyle, and dialogue choice context</li>
          <li>Variables are used in operations, conditions, and text fields</li>
          <li>Use either <code className="bg-white px-1 rounded">{'{context.var}'}</code> or <code className="bg-white px-1 rounded">context.var</code> format</li>
          <li>Ensure all used variables are set before they are referenced</li>
        </ul>
      </div>
    </div>
  );
}
