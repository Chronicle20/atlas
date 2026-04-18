'use client';

import React, { useState, useCallback } from 'react';
import { Trash2, Plus } from 'lucide-react';

export interface Operation {
  type: string;
  params: Record<string, string>;
}

interface OperationsEditorProps {
  operations: Operation[];
  onChange: (operations: Operation[]) => void;
}

// Operation type definitions with their parameters
const OPERATION_TYPES = {
  // Remote operations
  'award_item': {
    label: 'Award Item',
    category: 'Inventory',
    params: [
      { name: 'itemId', label: 'Item ID', required: true, type: 'number', placeholder: '4031013 or {context.itemId}' },
      { name: 'quantity', label: 'Quantity', required: true, type: 'number', placeholder: '1 or {context.quantity}' }
    ]
  },
  'award_mesos': {
    label: 'Award Mesos',
    category: 'Inventory',
    params: [
      { name: 'amount', label: 'Amount', required: true, type: 'number', placeholder: '1000 or {context.amount}' },
      { name: 'actorId', label: 'Actor ID', required: false, type: 'number', placeholder: 'NPC ID (optional)' },
      { name: 'actorType', label: 'Actor Type', required: false, type: 'text', placeholder: 'NPC (default)' }
    ]
  },
  'award_exp': {
    label: 'Award Experience',
    category: 'Character',
    params: [
      { name: 'amount', label: 'Amount', required: true, type: 'number', placeholder: '100 or {context.exp}' },
      { name: 'type', label: 'Type', required: false, type: 'text', placeholder: 'WHITE (default)' },
      { name: 'attr1', label: 'Attribute 1', required: false, type: 'number', placeholder: '0 (default)' }
    ]
  },
  'award_level': {
    label: 'Award Level',
    category: 'Character',
    params: [
      { name: 'amount', label: 'Levels', required: true, type: 'number', placeholder: '1 or {context.levels}' }
    ]
  },
  'warp_to_map': {
    label: 'Warp to Map',
    category: 'Movement',
    params: [
      { name: 'mapId', label: 'Map ID', required: true, type: 'number', placeholder: '100000000 or {context.mapId}' },
      { name: 'portalId', label: 'Portal ID', required: false, type: 'number', placeholder: '0 (default spawn)' }
    ]
  },
  'warp_to_random_portal': {
    label: 'Warp to Random Portal',
    category: 'Movement',
    params: [
      { name: 'mapId', label: 'Map ID', required: true, type: 'number', placeholder: '100000000 or {context.mapId}' }
    ]
  },
  'change_job': {
    label: 'Change Job',
    category: 'Character',
    params: [
      { name: 'jobId', label: 'Job ID', required: true, type: 'number', placeholder: '100 or {context.jobId}' }
    ]
  },
  'increase_buddy_capacity': {
    label: 'Increase Buddy Capacity',
    category: 'Social',
    params: [
      { name: 'amount', label: 'Amount', required: true, type: 'number', placeholder: '5 or {context.amount}' }
    ]
  },
  'create_skill': {
    label: 'Create Skill',
    category: 'Skills',
    params: [
      { name: 'skillId', label: 'Skill ID', required: true, type: 'number', placeholder: '1001 or {context.skillId}' },
      { name: 'level', label: 'Level', required: false, type: 'number', placeholder: '1 (default)' },
      { name: 'masterLevel', label: 'Master Level', required: false, type: 'number', placeholder: '1 (default)' }
    ]
  },
  'update_skill': {
    label: 'Update Skill',
    category: 'Skills',
    params: [
      { name: 'skillId', label: 'Skill ID', required: true, type: 'number', placeholder: '1001 or {context.skillId}' },
      { name: 'level', label: 'Level', required: false, type: 'number', placeholder: '1 (default)' },
      { name: 'masterLevel', label: 'Master Level', required: false, type: 'number', placeholder: '1 (default)' }
    ]
  },
  'destroy_item': {
    label: 'Destroy Item',
    category: 'Inventory',
    params: [
      { name: 'itemId', label: 'Item ID', required: true, type: 'number', placeholder: '4031013 or {context.itemId}' },
      { name: 'quantity', label: 'Quantity', required: true, type: 'number', placeholder: '1 or {context.quantity}' }
    ]
  },
  'gain_closeness': {
    label: 'Gain Pet Closeness',
    category: 'Pets',
    params: [
      { name: 'petId', label: 'Pet ID', required: false, type: 'number', placeholder: 'Pet ID or leave empty to use petIndex' },
      { name: 'petIndex', label: 'Pet Slot Index', required: false, type: 'number', placeholder: '0, 1, or 2 (if no petId)' },
      { name: 'amount', label: 'Amount', required: true, type: 'number', placeholder: '1 or {context.amount}' }
    ]
  },
  'change_hair': {
    label: 'Change Hair',
    category: 'Cosmetics',
    params: [
      { name: 'styleId', label: 'Hair Style ID', required: true, type: 'number', placeholder: '30000 or {context.selectedStyle}' }
    ]
  },
  'change_face': {
    label: 'Change Face',
    category: 'Cosmetics',
    params: [
      { name: 'styleId', label: 'Face Style ID', required: true, type: 'number', placeholder: '20000 or {context.selectedStyle}' }
    ]
  },
  'change_skin': {
    label: 'Change Skin',
    category: 'Cosmetics',
    params: [
      { name: 'styleId', label: 'Skin Color ID', required: true, type: 'number', placeholder: '0-13 or {context.selectedStyle}' }
    ]
  },
  // Local operations
  'local:log': {
    label: 'Log Message',
    category: 'Debug',
    params: [
      { name: 'message', label: 'Message', required: true, type: 'text', placeholder: 'Log message or {context.message}' }
    ]
  },
  'local:debug': {
    label: 'Debug Log',
    category: 'Debug',
    params: [
      { name: 'message', label: 'Message', required: true, type: 'text', placeholder: 'Debug message or {context.message}' }
    ]
  },
  'local:generate_hair_styles': {
    label: 'Generate Hair Styles',
    category: 'Cosmetics',
    params: [
      { name: 'baseStyles', label: 'Base Styles', required: false, type: 'text', placeholder: 'Comma-separated IDs (optional)' },
      { name: 'genderFilter', label: 'Gender Filter', required: false, type: 'text', placeholder: 'MALE, FEMALE, or ALL (default)' },
      { name: 'preserveColor', label: 'Preserve Color', required: false, type: 'text', placeholder: 'true or false' },
      { name: 'validateExists', label: 'Validate Exists', required: false, type: 'text', placeholder: 'true (default)' },
      { name: 'excludeEquipped', label: 'Exclude Equipped', required: false, type: 'text', placeholder: 'true (default)' },
      { name: 'outputContextKey', label: 'Output Context Key', required: false, type: 'text', placeholder: 'generatedStyles (default)' }
    ]
  },
  'local:generate_hair_colors': {
    label: 'Generate Hair Colors',
    category: 'Cosmetics',
    params: [
      { name: 'colors', label: 'Colors', required: false, type: 'text', placeholder: 'Comma-separated color IDs (optional)' },
      { name: 'outputContextKey', label: 'Output Context Key', required: false, type: 'text', placeholder: 'generatedColors (default)' }
    ]
  },
  'local:generate_face_styles': {
    label: 'Generate Face Styles',
    category: 'Cosmetics',
    params: [
      { name: 'baseStyles', label: 'Base Styles', required: false, type: 'text', placeholder: 'Comma-separated IDs (optional)' },
      { name: 'genderFilter', label: 'Gender Filter', required: false, type: 'text', placeholder: 'MALE, FEMALE, or ALL (default)' },
      { name: 'validateExists', label: 'Validate Exists', required: false, type: 'text', placeholder: 'true (default)' },
      { name: 'excludeEquipped', label: 'Exclude Equipped', required: false, type: 'text', placeholder: 'true (default)' },
      { name: 'outputContextKey', label: 'Output Context Key', required: false, type: 'text', placeholder: 'generatedFaces (default)' }
    ]
  },
  'local:select_random_cosmetic': {
    label: 'Select Random Cosmetic',
    category: 'Cosmetics',
    params: [
      { name: 'stylesContextKey', label: 'Styles Context Key', required: true, type: 'text', placeholder: 'generatedStyles' },
      { name: 'outputContextKey', label: 'Output Context Key', required: true, type: 'text', placeholder: 'selectedStyle' }
    ]
  },
  'local:fetch_map_player_counts': {
    label: 'Fetch Map Player Counts',
    category: 'Maps',
    params: [
      { name: 'mapIds', label: 'Map IDs', required: true, type: 'text', placeholder: 'Comma-separated map IDs or {context.mapIds}' }
    ]
  }
} as const;

const CATEGORIES = ['All', 'Inventory', 'Character', 'Movement', 'Social', 'Skills', 'Pets', 'Cosmetics', 'Maps', 'Debug'] as const;

export function OperationsEditor({ operations, onChange }: OperationsEditorProps) {
  const [selectedCategory, setSelectedCategory] = useState<string>('All');

  const handleAddOperation = useCallback(() => {
    const newOperation: Operation = {
      type: 'award_item',
      params: {}
    };
    onChange([...operations, newOperation]);
  }, [operations, onChange]);

  const handleRemoveOperation = useCallback((index: number) => {
    const newOperations = operations.filter((_, i) => i !== index);
    onChange(newOperations);
  }, [operations, onChange]);

  const handleOperationTypeChange = useCallback((index: number, newType: string) => {
    const newOperations = [...operations];
    newOperations[index] = {
      type: newType,
      params: {}
    };
    onChange(newOperations);
  }, [operations, onChange]);

  const handleParamChange = useCallback((operationIndex: number, paramName: string, value: string) => {
    const newOperations = [...operations];
    const currentOp = newOperations[operationIndex]!;
    newOperations[operationIndex] = {
      type: currentOp.type,
      params: {
        ...currentOp.params,
        [paramName]: value
      }
    };
    onChange(newOperations);
  }, [operations, onChange]);

  const filteredOperationTypes = Object.entries(OPERATION_TYPES).filter(([_, config]) => {
    if (selectedCategory === 'All') return true;
    return config.category === selectedCategory;
  });

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h3 className="text-lg font-semibold">Operations</h3>
        <button
          type="button"
          onClick={handleAddOperation}
          className="flex items-center gap-1 px-3 py-1.5 text-sm bg-blue-600 text-white rounded hover:bg-blue-700"
        >
          <Plus className="h-4 w-4" />
          Add Operation
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

      {operations.length === 0 ? (
        <div className="text-sm text-gray-500 italic p-4 border border-dashed border-gray-300 rounded text-center">
          No operations defined. Click &quot;Add Operation&quot; to create one.
        </div>
      ) : (
        <div className="space-y-3">
          {operations.map((operation, operationIndex) => {
            const operationConfig = OPERATION_TYPES[operation.type as keyof typeof OPERATION_TYPES];

            return (
              <div key={operationIndex} className="border border-gray-300 rounded p-3 space-y-3 bg-gray-50">
                <div className="flex items-start justify-between gap-2">
                  <div className="flex-1">
                    <label className="block text-xs font-medium text-gray-700 mb-1">
                      Operation Type
                    </label>
                    <select
                      value={operation.type}
                      onChange={(e) => handleOperationTypeChange(operationIndex, e.target.value)}
                      className="w-full px-2 py-1.5 text-sm border border-gray-300 rounded focus:ring-blue-500 focus:border-blue-500"
                    >
                      {filteredOperationTypes.map(([type, config]) => (
                        <option key={type} value={type}>
                          {config.label} ({config.category})
                        </option>
                      ))}
                    </select>
                  </div>
                  <button
                    type="button"
                    onClick={() => handleRemoveOperation(operationIndex)}
                    className="mt-6 p-1.5 text-red-600 hover:bg-red-50 rounded"
                    title="Remove operation"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>

                {operationConfig && (
                  <div className="space-y-2 pl-2 border-l-2 border-blue-200">
                    {operationConfig.params.map((param) => (
                      <div key={param.name}>
                        <label className="block text-xs font-medium text-gray-700 mb-1">
                          {param.label}
                          {param.required && <span className="text-red-500 ml-1">*</span>}
                        </label>
                        <input
                          type="text"
                          value={operation.params[param.name] || ''}
                          onChange={(e) => handleParamChange(operationIndex, param.name, e.target.value)}
                          placeholder={param.placeholder}
                          className="w-full px-2 py-1.5 text-sm border border-gray-300 rounded focus:ring-blue-500 focus:border-blue-500"
                        />
                      </div>
                    ))}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}

      <div className="text-xs text-gray-600 bg-blue-50 p-3 rounded border border-blue-200">
        <p className="font-semibold mb-1">Context Variables:</p>
        <p>
          Use <code className="bg-white px-1 rounded">{'context.variableName'}</code> or{' '}
          <code className="bg-white px-1 rounded">{'{context.variableName}'}</code> to reference
          values from the conversation context.
        </p>
      </div>
    </div>
  );
}
