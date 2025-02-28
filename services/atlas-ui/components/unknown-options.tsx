import {FieldValues, Path, PathValue, UseFormReturn, useWatch} from "react-hook-form";
import {useState} from "react";
import {FormLabel} from "@/components/ui/form";
import {Button} from "@/components/ui/button";
import {Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle} from "@/components/ui/dialog";
import {Input} from "@/components/ui/input";
import {Plus, X} from "lucide-react";

interface OptionsFieldProps<T extends FieldValues> {
    form: UseFormReturn<T>;
    path: Path<T>;
}

export function OptionsField<T extends FieldValues>({form, path}: OptionsFieldProps<T>) {
    const options = useWatch({control: form.control, name: path}) as Record<string, string> | undefined;

    const [option, setOption] = useState("");
    const [newKey, setNewKey] = useState("");
    const [newValue, setNewValue] = useState("");
    const [isOptionsDialogOpen, setDialogOptionsOpen] = useState(false);
    const [isOptionChildDialogOpen, setDialogOptionChildOpen] = useState(false);

    const handleAddOption = (key: string) => {
        const currentOptions = form.getValues(path) || {};
        setNewValue("");
        form.setValue(path, {...currentOptions, [key]: ""} as PathValue<T, Path<T>>);
    };

    const handleRemoveOption = (key: string) => {
        const updatedOptions = {...form.getValues(path)};
        delete updatedOptions[key];
        if (Object.entries(updatedOptions).length === 0) {
            form.setValue(path, undefined);
        } else {
            form.setValue(path, updatedOptions);
        }
    };

    const handleAddOptionChild = (opt: string, key: string, value: string) => {
        const currentOptions = form.getValues(path) || {};
        const updatedOptions = {
            ...currentOptions,
            [opt]: {
                ...currentOptions[opt],
                [key]: value
            }
        };
        setOption("");
        setNewKey("");
        setNewValue("");
        form.setValue(path, updatedOptions as PathValue<T, Path<T>>);
    };

    const handleRemoveOptionChild = (opt: string, key: string) => {
        const updatedOptions = {...form.getValues(path)};
        delete updatedOptions[opt][key];
        form.setValue(path, updatedOptions);
    };

    if (options === null || typeof options !== "object") {
        return (
            <div className="flex flex-col gap-2">
                <div className="space-y-2">
                    <FormLabel>Options</FormLabel>
                    <div className="border p-4 rounded-md gap-2">
                        No options configured.
                    </div>
                </div>
                <div className="flex flex-row gap-2 justify-between">
                    <Button type="button" onClick={() => setDialogOptionsOpen(true)}>
                        Add
                    </Button>
                </div>
                <Dialog open={isOptionsDialogOpen} onOpenChange={setDialogOptionsOpen}>
                    <DialogContent>
                        <DialogHeader>
                            <DialogTitle>Add a New Value</DialogTitle>
                        </DialogHeader>
                        <Input
                            value={newValue}
                            onChange={(e) => setNewValue(e.target.value)}
                            placeholder="Enter value..."
                        />
                        <DialogFooter>
                            <Button variant="outline" onClick={() => setDialogOptionsOpen(false)}>Cancel</Button>
                            <Button onClick={() => {
                                handleAddOption(newValue);
                                setDialogOptionsOpen(false);
                            }}>Add</Button>
                        </DialogFooter>
                    </DialogContent>
                </Dialog>
            </div>
        )
    }

    return (
        <div className="flex flex-col gap-2">
            <div className="space-y-2">
                <FormLabel>Options</FormLabel>
                <div className="flex flex-col gap-2">
                    {Object.entries(options).map(([opt, optValue]) => (
                        <div key={opt} className="relative border p-4 rounded-md gap-2">
                            <FormLabel>{opt}</FormLabel>
                            <div className="flex flex-row flex-wrap p-2 justify-start gap-2">
                                {Object.entries(optValue)
                                    .sort(([keyA, valA], [keyB, valB]) => (valA > valB ? 1 : -1))
                                    .map(([key, value]) => (
                                        <div key={key} className="flex items-center">
                                            <Button type="button" variant="outline"
                                                    onClick={() => handleRemoveOptionChild(opt, key)}>
                                                {key} : {String(value)}
                                                <X/>
                                            </Button>
                                        </div>
                                    ))}
                                <Button type="button" variant="outline" size="icon" onClick={() => {
                                    setOption(opt);
                                    setDialogOptionChildOpen(true);
                                }}>
                                    <Plus/>
                                </Button>
                            </div>
                            <Button type="button" className="absolute top-0 right-0" variant="ghost" size="icon"
                                    onClick={() => handleRemoveOption(opt)}>
                                <X/>
                            </Button>
                        </div>
                    ))}
                </div>
                <div className="flex flex-row gap-2 justify-between">
                    <Button type="button" onClick={() => setDialogOptionsOpen(true)}>
                        Add
                    </Button>
                </div>
            </div>
            <Dialog open={isOptionsDialogOpen} onOpenChange={setDialogOptionsOpen}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Add a New Value</DialogTitle>
                    </DialogHeader>
                    <Input
                        value={newValue}
                        onChange={(e) => setNewValue(e.target.value)}
                        placeholder="Enter value..."
                    />
                    <DialogFooter>
                        <Button variant="outline" onClick={() => {
                            setNewValue("");
                            setDialogOptionsOpen(false)
                        }}>Cancel</Button>
                        <Button onClick={() => {
                            handleAddOption(newValue);
                            setDialogOptionsOpen(false);
                        }}>Add</Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
            <Dialog open={isOptionChildDialogOpen} onOpenChange={setDialogOptionChildOpen}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Add a New Key Value Pair for {option}</DialogTitle>
                    </DialogHeader>
                    <Input
                        value={newKey}
                        onChange={(e) => setNewKey(e.target.value)}
                        placeholder="Enter key..."
                    />
                    <Input
                        value={newValue}
                        onChange={(e) => setNewValue(e.target.value)}
                        placeholder="Enter value..."
                    />
                    <DialogFooter>
                        <Button variant="outline" onClick={() => {
                            setOption("");
                            setNewKey("");
                            setNewValue("");
                            setDialogOptionChildOpen(false)
                        }}>Cancel</Button>
                        <Button onClick={() => {
                            handleAddOptionChild(option, newKey, newValue);
                            setDialogOptionChildOpen(false);
                        }}>Add</Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    )
}