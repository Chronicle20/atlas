'use client';

import React, { useState, useCallback } from 'react';
import { Trash2, Plus } from 'lucide-react';

export interface Condition {
  type: string;
  operator: string;
  value: string;
  referenceId?: string;
  step?: string;
  worldId?: number | string;
  channelId?: number | string;
}

export interface Outcome {
  conditions: Condition[];
  nextState: string;
}

interface ConditionsBuilderProps {
  outcomes: Outcome[];
  onChange: (outcomes: Outcome[]) => void;
  availableStates: string[];
}

// Condition field names
type ConditionField = 'value' | 'referenceId' | 'step' | 'worldId' | 'channelId';

// Condition type definitions
const CONDITION_TYPES: Record<string, {
  label: string;
  category: string;
  description: string;
  fields: ConditionField[];
  supportsContext: boolean;
}> = {
  'level': {
    label: 'Character Level',
    category: 'Character',
    description: 'Check character level',
    fields: ['value'],
    supportsContext: true
  },
  'jobId': {
    label: 'Job ID',
    category: 'Character',
    description: 'Check character job',
    fields: ['value'],
    supportsContext: true
  },
  'meso': {
    label: 'Mesos',
    category: 'Currency',
    description: 'Check meso amount',
    fields: ['value'],
    supportsContext: true
  },
  'fame': {
    label: 'Fame',
    category: 'Character',
    description: 'Check fame amount',
    fields: ['value'],
    supportsContext: true
  },
  'buddyCapacity': {
    label: 'Buddy Capacity',
    category: 'Social',
    description: 'Check buddy list capacity',
    fields: ['value'],
    supportsContext: true
  },
  'item': {
    label: 'Item',
    category: 'Inventory',
    description: 'Check item quantity',
    fields: ['value', 'referenceId'],
    supportsContext: true
  },
  'mapId': {
    label: 'Map ID',
    category: 'Location',
    description: 'Check current map',
    fields: ['value'],
    supportsContext: true
  },
  'mapCapacity': {
    label: 'Map Capacity',
    category: 'Location',
    description: 'Check map player count',
    fields: ['value', 'worldId', 'channelId'],
    supportsContext: true
  },
  'questStatus': {
    label: 'Quest Status',
    category: 'Quests',
    description: 'Check quest status',
    fields: ['value', 'referenceId', 'step'],
    supportsContext: true
  }
};

const OPERATORS = [
  { value: '==', label: '== (equals)' },
  { value: '>=', label: '>= (greater than or equal)' },
  { value: '<=', label: '<= (less than or equal)' },
  { value: '>', label: '> (greater than)' },
  { value: '<', label: '< (less than)' }
] as const;

const CATEGORIES = ['All', 'Character', 'Currency', 'Inventory', 'Location', 'Social', 'Quests'] as const;

export function ConditionsBuilder({ outcomes, onChange, availableStates }: ConditionsBuilderProps) {
  const [selectedCategory, setSelectedCategory] = useState<string>('All');

  const handleAddOutcome = useCallback(() => {
    const newOutcome: Outcome = {
      conditions: [],
      nextState: availableStates[0] || ''
    };
    onChange([...outcomes, newOutcome]);
  }, [outcomes, onChange, availableStates]);

  const handleRemoveOutcome = useCallback((index: number) => {
    const newOutcomes = outcomes.filter((_, i) => i !== index);
    onChange(newOutcomes);
  }, [outcomes, onChange]);

  const handleAddCondition = useCallback((outcomeIndex: number) => {
    const newOutcomes = [...outcomes];
    const newCondition: Condition = {
      type: 'level',
      operator: '>=',
      value: ''
    };
    const currentOutcome = newOutcomes[outcomeIndex]!;
    newOutcomes[outcomeIndex] = {
      conditions: [...currentOutcome.conditions, newCondition],
      nextState: currentOutcome.nextState
    };
    onChange(newOutcomes);
  }, [outcomes, onChange]);

  const handleRemoveCondition = useCallback((outcomeIndex: number, conditionIndex: number) => {
    const newOutcomes = [...outcomes];
    const currentOutcome = newOutcomes[outcomeIndex]!;
    newOutcomes[outcomeIndex] = {
      conditions: currentOutcome.conditions.filter((_, i) => i !== conditionIndex),
      nextState: currentOutcome.nextState
    };
    onChange(newOutcomes);
  }, [outcomes, onChange]);

  const handleConditionChange = useCallback((outcomeIndex: number, conditionIndex: number, field: string, value: string) => {
    const newOutcomes = [...outcomes];
    const condition = { ...newOutcomes[outcomeIndex]!.conditions[conditionIndex]! };

    if (field === 'type') {
      // Reset condition when type changes
      condition.type = value;
      condition.value = '';
      delete condition.referenceId;
      delete condition.step;
      delete condition.worldId;
      delete condition.channelId;
    } else {
      (condition as any)[field] = value;
    }

    newOutcomes[outcomeIndex]!.conditions[conditionIndex] = condition;
    onChange(newOutcomes);
  }, [outcomes, onChange]);

  const handleNextStateChange = useCallback((outcomeIndex: number, nextState: string) => {
    const newOutcomes = [...outcomes];
    const currentOutcome = newOutcomes[outcomeIndex]!;
    newOutcomes[outcomeIndex] = {
      conditions: currentOutcome.conditions,
      nextState
    };
    onChange(newOutcomes);
  }, [outcomes, onChange]);

  const filteredConditionTypes = Object.entries(CONDITION_TYPES).filter(([_, config]) => {
    if (selectedCategory === 'All') return true;
    return config.category === selectedCategory;
  });

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h3 className="text-lg font-semibold">Outcomes</h3>
        <button
          type="button"
          onClick={handleAddOutcome}
          className="flex items-center gap-1 px-3 py-1.5 text-sm bg-blue-600 text-white rounded hover:bg-blue-700"
        >
          <Plus className="h-4 w-4" />
          Add Outcome
        </button>
      </div>

      <div className="flex gap-2 flex-wrap">
        {CATEGORIES.map(category => (
          <button
            key={category}
            type="button"
            onClick={() => setSelectedCategory(category)}
            className={`px-3 py-1 text-xs rounded ${
              selectedCategory === category
                ? 'bg-blue-600 text-white'
                : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
            }`}
          >
            {category}
          </button>
        ))}
      </div>

      {outcomes.length === 0 ? (
        <div className="text-sm text-gray-500 italic p-4 border border-dashed border-gray-300 rounded text-center">
          No outcomes defined. Click &quot;Add Outcome&quot; to create one.
        </div>
      ) : (
        <div className="space-y-4">
          {outcomes.map((outcome, outcomeIndex) => (
            <div key={outcomeIndex} className="border border-gray-400 rounded p-4 space-y-3 bg-gray-50">
              <div className="flex items-start justify-between gap-2">
                <div className="flex-1">
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Outcome {outcomeIndex + 1}
                  </label>
                </div>
                <button
                  type="button"
                  onClick={() => handleRemoveOutcome(outcomeIndex)}
                  className="p-1.5 text-red-600 hover:bg-red-50 rounded"
                  title="Remove outcome"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>

              {/* Next State */}
              <div>
                <label className="block text-xs font-medium text-gray-700 mb-1">
                  Next State <span className="text-red-500">*</span>
                </label>
                <select
                  value={outcome.nextState}
                  onChange={(e) => handleNextStateChange(outcomeIndex, e.target.value)}
                  className="w-full px-2 py-1.5 text-sm border border-gray-300 rounded focus:ring-blue-500 focus:border-blue-500"
                >
                  <option value="">-- Select State --</option>
                  {availableStates.map((stateId) => (
                    <option key={stateId} value={stateId}>
                      {stateId}
                    </option>
                  ))}
                </select>
              </div>

              {/* Conditions */}
              <div className="space-y-2">
                <div className="flex justify-between items-center">
                  <label className="block text-sm font-medium text-gray-700">
                    Conditions {outcome.conditions.length > 0 && `(${outcome.conditions.length})`}
                  </label>
                  <button
                    type="button"
                    onClick={() => handleAddCondition(outcomeIndex)}
                    className="flex items-center gap-1 px-2 py-1 text-xs bg-green-600 text-white rounded hover:bg-green-700"
                  >
                    <Plus className="h-3 w-3" />
                    Add Condition
                  </button>
                </div>

                {outcome.conditions.length === 0 ? (
                  <div className="text-xs text-gray-500 italic p-2 border border-dashed border-gray-300 rounded text-center">
                    No conditions. This outcome will be used as default/fallback.
                  </div>
                ) : (
                  <div className="space-y-2 pl-3 border-l-2 border-green-200">
                    {outcome.conditions.map((condition, conditionIndex) => {
                      const conditionConfig = CONDITION_TYPES[condition.type as keyof typeof CONDITION_TYPES];

                      return (
                        <div key={conditionIndex} className="border border-gray-300 rounded p-2 space-y-2 bg-white">
                          <div className="flex items-start justify-between gap-2">
                            <div className="flex-1 grid grid-cols-2 gap-2">
                              {/* Condition Type */}
                              <div>
                                <label className="block text-xs font-medium text-gray-700 mb-1">
                                  Type <span className="text-red-500">*</span>
                                </label>
                                <select
                                  value={condition.type}
                                  onChange={(e) => handleConditionChange(outcomeIndex, conditionIndex, 'type', e.target.value)}
                                  className="w-full px-2 py-1 text-xs border border-gray-300 rounded focus:ring-blue-500 focus:border-blue-500"
                                >
                                  {filteredConditionTypes.map(([type, config]) => (
                                    <option key={type} value={type}>
                                      {config.label} ({config.category})
                                    </option>
                                  ))}
                                </select>
                              </div>

                              {/* Operator */}
                              <div>
                                <label className="block text-xs font-medium text-gray-700 mb-1">
                                  Operator <span className="text-red-500">*</span>
                                </label>
                                <select
                                  value={condition.operator}
                                  onChange={(e) => handleConditionChange(outcomeIndex, conditionIndex, 'operator', e.target.value)}
                                  className="w-full px-2 py-1 text-xs border border-gray-300 rounded focus:ring-blue-500 focus:border-blue-500"
                                >
                                  {OPERATORS.map((op) => (
                                    <option key={op.value} value={op.value}>
                                      {op.label}
                                    </option>
                                  ))}
                                </select>
                              </div>
                            </div>
                            <button
                              type="button"
                              onClick={() => handleRemoveCondition(outcomeIndex, conditionIndex)}
                              className="mt-5 p-1 text-red-600 hover:bg-red-50 rounded"
                              title="Remove condition"
                            >
                              <Trash2 className="h-3 w-3" />
                            </button>
                          </div>

                          {/* Condition-specific fields */}
                          {conditionConfig && (
                            <div className="grid grid-cols-2 gap-2">
                              {/* Value */}
                              <div>
                                <label className="block text-xs font-medium text-gray-700 mb-1">
                                  Value <span className="text-red-500">*</span>
                                </label>
                                <input
                                  type="text"
                                  value={condition.value}
                                  onChange={(e) => handleConditionChange(outcomeIndex, conditionIndex, 'value', e.target.value)}
                                  placeholder="10 or {context.minLevel}"
                                  className="w-full px-2 py-1 text-xs border border-gray-300 rounded focus:ring-blue-500 focus:border-blue-500"
                                />
                              </div>

                              {/* ReferenceId (for item, questStatus) */}
                              {conditionConfig.fields.includes('referenceId') && (
                                <div>
                                  <label className="block text-xs font-medium text-gray-700 mb-1">
                                    Reference ID {(condition.type === 'item' || condition.type === 'questStatus') && <span className="text-red-500">*</span>}
                                  </label>
                                  <input
                                    type="text"
                                    value={condition.referenceId || ''}
                                    onChange={(e) => handleConditionChange(outcomeIndex, conditionIndex, 'referenceId', e.target.value)}
                                    placeholder={condition.type === 'item' ? 'Item ID' : 'Quest ID'}
                                    className="w-full px-2 py-1 text-xs border border-gray-300 rounded focus:ring-blue-500 focus:border-blue-500"
                                  />
                                </div>
                              )}

                              {/* Step (for questStatus) */}
                              {conditionConfig.fields.includes('step') && (
                                <div>
                                  <label className="block text-xs font-medium text-gray-700 mb-1">
                                    Quest Step
                                  </label>
                                  <input
                                    type="text"
                                    value={condition.step || ''}
                                    onChange={(e) => handleConditionChange(outcomeIndex, conditionIndex, 'step', e.target.value)}
                                    placeholder="Optional quest step"
                                    className="w-full px-2 py-1 text-xs border border-gray-300 rounded focus:ring-blue-500 focus:border-blue-500"
                                  />
                                </div>
                              )}

                              {/* WorldId (for mapCapacity) */}
                              {conditionConfig.fields.includes('worldId') && (
                                <div>
                                  <label className="block text-xs font-medium text-gray-700 mb-1">
                                    World ID
                                  </label>
                                  <input
                                    type="text"
                                    value={condition.worldId || ''}
                                    onChange={(e) => handleConditionChange(outcomeIndex, conditionIndex, 'worldId', e.target.value)}
                                    placeholder="0 or {context.worldId}"
                                    className="w-full px-2 py-1 text-xs border border-gray-300 rounded focus:ring-blue-500 focus:border-blue-500"
                                  />
                                </div>
                              )}

                              {/* ChannelId (for mapCapacity) */}
                              {conditionConfig.fields.includes('channelId') && (
                                <div>
                                  <label className="block text-xs font-medium text-gray-700 mb-1">
                                    Channel ID
                                  </label>
                                  <input
                                    type="text"
                                    value={condition.channelId || ''}
                                    onChange={(e) => handleConditionChange(outcomeIndex, conditionIndex, 'channelId', e.target.value)}
                                    placeholder="0 or {context.channelId}"
                                    className="w-full px-2 py-1 text-xs border border-gray-300 rounded focus:ring-blue-500 focus:border-blue-500"
                                  />
                                </div>
                              )}
                            </div>
                          )}

                          <div className="text-xs text-gray-600">
                            <span className="font-semibold">{conditionConfig?.label}:</span> {conditionConfig?.description}
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      <div className="text-xs text-gray-600 bg-blue-50 p-3 rounded border border-blue-200">
        <p className="font-semibold mb-1">Outcomes & Conditions:</p>
        <ul className="list-disc list-inside space-y-1">
          <li>Outcomes are evaluated in order. The first matching outcome determines the next state.</li>
          <li>An outcome with no conditions acts as a default/fallback.</li>
          <li>All conditions within an outcome must pass (AND logic).</li>
          <li>Use <code className="bg-white px-1 rounded">{'{context.variableName}'}</code> for dynamic values.</li>
        </ul>
      </div>
    </div>
  );
}
