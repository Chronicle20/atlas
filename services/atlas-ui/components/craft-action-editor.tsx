'use client';

import React, { useCallback } from 'react';
import { Trash2, Plus } from 'lucide-react';

export interface CraftActionData {
  itemId: number;
  materials: number[];
  quantities: number[];
  mesoCost: number;
  stimulatorId?: number;
  stimulatorFailChance?: number;
  successState: string;
  failureState: string;
  missingMaterialsState: string;
}

interface CraftActionEditorProps {
  data: CraftActionData;
  onChange: (data: CraftActionData) => void;
  availableStates: string[];
}

export function CraftActionEditor({ data, onChange, availableStates }: CraftActionEditorProps) {
  const handleFieldChange = useCallback((field: keyof CraftActionData, value: any) => {
    onChange({
      ...data,
      [field]: value
    });
  }, [data, onChange]);

  const handleAddMaterial = useCallback(() => {
    onChange({
      ...data,
      materials: [...data.materials, 0],
      quantities: [...data.quantities, 1]
    });
  }, [data, onChange]);

  const handleRemoveMaterial = useCallback((index: number) => {
    const newMaterials = data.materials.filter((_, i) => i !== index);
    const newQuantities = data.quantities.filter((_, i) => i !== index);
    onChange({
      ...data,
      materials: newMaterials,
      quantities: newQuantities
    });
  }, [data, onChange]);

  const handleMaterialChange = useCallback((index: number, value: number) => {
    const newMaterials = [...data.materials];
    newMaterials[index] = value;
    onChange({
      ...data,
      materials: newMaterials
    });
  }, [data, onChange]);

  const handleQuantityChange = useCallback((index: number, value: number) => {
    const newQuantities = [...data.quantities];
    newQuantities[index] = value;
    onChange({
      ...data,
      quantities: newQuantities
    });
  }, [data, onChange]);

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h3 className="text-lg font-semibold">Craft Action Parameters</h3>
      </div>

      <div className="text-sm text-gray-600 bg-blue-50 p-3 rounded border border-blue-200">
        <p className="font-semibold mb-1">About Craft Actions:</p>
        <p>
          Craft actions allow players to create items by consuming materials and mesos.
          The action validates that the player has the required materials and mesos before crafting.
        </p>
      </div>

      {/* Result Item */}
      <div className="space-y-2">
        <label className="block text-sm font-medium text-gray-700">
          Result Item ID <span className="text-red-500">*</span>
        </label>
        <input
          type="number"
          value={data.itemId}
          onChange={(e) => handleFieldChange('itemId', parseInt(e.target.value) || 0)}
          placeholder="Item ID (e.g., 1302000)"
          className="w-full px-3 py-2 text-sm border border-gray-300 rounded focus:ring-blue-500 focus:border-blue-500"
        />
        <p className="text-xs text-gray-500">The item that will be crafted</p>
      </div>

      {/* Materials */}
      <div className="space-y-2">
        <div className="flex justify-between items-center">
          <label className="block text-sm font-medium text-gray-700">
            Required Materials <span className="text-red-500">*</span>
          </label>
          <button
            type="button"
            onClick={handleAddMaterial}
            className="flex items-center gap-1 px-2 py-1 text-xs bg-green-600 text-white rounded hover:bg-green-700"
          >
            <Plus className="h-3 w-3" />
            Add Material
          </button>
        </div>

        {data.materials.length === 0 ? (
          <div className="text-sm text-gray-500 italic p-4 border border-dashed border-gray-300 rounded text-center">
            No materials defined. Click "Add Material" to add one.
          </div>
        ) : (
          <div className="space-y-2">
            {data.materials.map((materialId, index) => (
              <div key={index} className="flex gap-2 items-start border border-gray-300 rounded p-2 bg-gray-50">
                <div className="flex-1 grid grid-cols-2 gap-2">
                  <div>
                    <label className="block text-xs font-medium text-gray-700 mb-1">
                      Material ID
                    </label>
                    <input
                      type="number"
                      value={materialId}
                      onChange={(e) => handleMaterialChange(index, parseInt(e.target.value) || 0)}
                      placeholder="Item ID"
                      className="w-full px-2 py-1 text-xs border border-gray-300 rounded focus:ring-blue-500 focus:border-blue-500"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-gray-700 mb-1">
                      Quantity
                    </label>
                    <input
                      type="number"
                      value={data.quantities[index]}
                      onChange={(e) => handleQuantityChange(index, parseInt(e.target.value) || 1)}
                      placeholder="Quantity"
                      min="1"
                      className="w-full px-2 py-1 text-xs border border-gray-300 rounded focus:ring-blue-500 focus:border-blue-500"
                    />
                  </div>
                </div>
                <button
                  type="button"
                  onClick={() => handleRemoveMaterial(index)}
                  className="mt-5 p-1 text-red-600 hover:bg-red-50 rounded"
                  title="Remove material"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Meso Cost */}
      <div className="space-y-2">
        <label className="block text-sm font-medium text-gray-700">
          Meso Cost
        </label>
        <input
          type="number"
          value={data.mesoCost}
          onChange={(e) => handleFieldChange('mesoCost', parseInt(e.target.value) || 0)}
          placeholder="Meso cost (e.g., 10000)"
          min="0"
          className="w-full px-3 py-2 text-sm border border-gray-300 rounded focus:ring-blue-500 focus:border-blue-500"
        />
        <p className="text-xs text-gray-500">The amount of mesos required to craft</p>
      </div>

      {/* Stimulator (Optional) */}
      <div className="border-t pt-4 space-y-3">
        <h4 className="text-sm font-semibold text-gray-700">Stimulator (Optional)</h4>
        <p className="text-xs text-gray-600">
          Stimulators can be used to enhance the crafted item, but have a chance to fail.
        </p>

        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-700">
              Stimulator Item ID
            </label>
            <input
              type="number"
              value={data.stimulatorId || ''}
              onChange={(e) => handleFieldChange('stimulatorId', parseInt(e.target.value) || undefined)}
              placeholder="Optional stimulator ID"
              className="w-full px-3 py-2 text-sm border border-gray-300 rounded focus:ring-blue-500 focus:border-blue-500"
            />
          </div>

          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-700">
              Fail Chance (%)
            </label>
            <input
              type="number"
              value={data.stimulatorFailChance || ''}
              onChange={(e) => handleFieldChange('stimulatorFailChance', parseInt(e.target.value) || undefined)}
              placeholder="Fail chance (0-100)"
              min="0"
              max="100"
              className="w-full px-3 py-2 text-sm border border-gray-300 rounded focus:ring-blue-500 focus:border-blue-500"
            />
          </div>
        </div>
      </div>

      {/* State Transitions */}
      <div className="border-t pt-4 space-y-3">
        <h4 className="text-sm font-semibold text-gray-700">State Transitions</h4>
        <p className="text-xs text-gray-600">
          Define which states to transition to based on the craft result.
        </p>

        <div className="space-y-3">
          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-700">
              Success State <span className="text-red-500">*</span>
            </label>
            <select
              value={data.successState}
              onChange={(e) => handleFieldChange('successState', e.target.value)}
              className="w-full px-3 py-2 text-sm border border-gray-300 rounded focus:ring-blue-500 focus:border-blue-500"
            >
              <option value="">-- Select State --</option>
              {availableStates.map((stateId) => (
                <option key={stateId} value={stateId}>
                  {stateId}
                </option>
              ))}
            </select>
            <p className="text-xs text-gray-500">State to transition to when crafting succeeds</p>
          </div>

          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-700">
              Failure State <span className="text-red-500">*</span>
            </label>
            <select
              value={data.failureState}
              onChange={(e) => handleFieldChange('failureState', e.target.value)}
              className="w-full px-3 py-2 text-sm border border-gray-300 rounded focus:ring-blue-500 focus:border-blue-500"
            >
              <option value="">-- Select State --</option>
              {availableStates.map((stateId) => (
                <option key={stateId} value={stateId}>
                  {stateId}
                </option>
              ))}
            </select>
            <p className="text-xs text-gray-500">State to transition to when stimulator fails (if using stimulator)</p>
          </div>

          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-700">
              Missing Materials State <span className="text-red-500">*</span>
            </label>
            <select
              value={data.missingMaterialsState}
              onChange={(e) => handleFieldChange('missingMaterialsState', e.target.value)}
              className="w-full px-3 py-2 text-sm border border-gray-300 rounded focus:ring-blue-500 focus:border-blue-500"
            >
              <option value="">-- Select State --</option>
              {availableStates.map((stateId) => (
                <option key={stateId} value={stateId}>
                  {stateId}
                </option>
              ))}
            </select>
            <p className="text-xs text-gray-500">State to transition to when player lacks required materials/mesos</p>
          </div>
        </div>
      </div>
    </div>
  );
}
