import {FieldValues, Path, PathValue, UseFormReturn, useWatch} from "react-hook-form";
import {useState} from "react";
import {FormLabel} from "@/components/ui/form";
import {Button} from "@/components/ui/button";
import {Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle} from "@/components/ui/dialog";
import {Input} from "@/components/ui/input";
import {Plus, X} from "lucide-react";
import {Switch} from "@/components/ui/switch";

interface OptionsFieldProps<T extends FieldValues> {
    form: UseFormReturn<T>;
    path: Path<T>;
}

export function OptionsField<T extends FieldValues>({form, path}: OptionsFieldProps<T>) {
    const options = useWatch({control: form.control, name: path}) as Record<string, string> | undefined;

    const [option, setOption] = useState("");
    const [newKey, setNewKey] = useState("");
    const [newValue, setNewValue] = useState("");
    const [isComplex, setIsComplex] = useState(false);
    const [complexProperties, setComplexProperties] = useState<{ key: string; value: string }[]>([]);
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
            form.setValue(path, undefined as PathValue<T, Path<T>>);
        } else {
            form.setValue(path, updatedOptions);
        }
    };

    const handleAddOptionChild = (opt: string, key: string, value: string) => {
        const currentOptions = form.getValues(path) || {};
        let parsedValue;

        if (isComplex) {
            parsedValue = complexProperties.reduce((obj, prop) => {
                obj[prop.key] = prop.value;
                return obj;
            }, {} as Record<string, string>);
        } else {
            parsedValue = isNaN(Number(value)) ? value : Number(value);
        }

        const updatedOptions = {
            ...currentOptions,
            [opt]: {
                // @ts-expect-error just let me do magic
                ...currentOptions[opt],
                [key]: parsedValue
            }
        };

        resetForm();
        form.setValue(path, updatedOptions as PathValue<T, Path<T>>);
    };

    const resetForm = () => {
        setNewKey("");
        setNewValue("");
        setComplexProperties([]);
        setIsComplex(false);
    };

    const handleRemoveOptionChild = (opt: string, key: string) => {
        const updatedOptions = { ...form.getValues(path) };
        delete updatedOptions[opt][key];
        if (Object.keys(updatedOptions[opt]).length === 0) {
            delete updatedOptions[opt];
        }
        form.setValue(path, updatedOptions);
    };

    if (!options || typeof options !== "object") {
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
                                    .sort(([, valA], [, valB]) => (valA > valB ? 1 : -1))
                                    .map(([key, value]) => (
                                        <div key={key} className="flex items-center">
                                            <Button type="button" variant="outline"
                                                    onClick={() => handleRemoveOptionChild(opt, key)}>
                                                {key} : {typeof value === "object" && value !== null
                                                ? Object.entries(value).map(([k, v]) => `${k}: ${String(v)}`).join(", ")
                                                : String(value)}
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
                        <DialogTitle>Add Key-Value for {option}</DialogTitle>
                    </DialogHeader>
                    <Input value={newKey} onChange={(e) => setNewKey(e.target.value)} placeholder="Enter key..." />
                    <Switch checked={isComplex} onCheckedChange={setIsComplex}>Complex Object</Switch>

                    {isComplex ? (
                        <>
                            {complexProperties.map((prop, index) => (
                                <div key={index} className="flex gap-2 items-center">
                                    <Input
                                        value={prop.key}
                                        onChange={(e) => {
                                            const newProps = [...complexProperties];
                                            newProps[index].key = e.target.value;
                                            setComplexProperties(newProps);
                                        }}
                                        placeholder="Property name..."
                                    />
                                    <Input
                                        value={prop.value}
                                        onChange={(e) => {
                                            const newProps = [...complexProperties];
                                            newProps[index].value = e.target.value;
                                            setComplexProperties(newProps);
                                        }}
                                        placeholder="Property value..."
                                    />
                                    <Button variant="ghost" onClick={() => setComplexProperties(complexProperties.filter((_, i) => i !== index))}>
                                        <X />
                                    </Button>
                                </div>
                            ))}
                            <Button onClick={() => setComplexProperties([...complexProperties, { key: "", value: "" }])}>
                                Add Property
                            </Button>
                        </>
                    ) : (
                        <Input value={newValue} onChange={(e) => setNewValue(e.target.value)} placeholder="Enter value..." />
                    )}
                    <DialogFooter>
                        <Button variant="outline" onClick={() => { resetForm(); setDialogOptionChildOpen(false); }}>Cancel</Button>
                        <Button onClick={() => { handleAddOptionChild(option, newKey, newValue); setDialogOptionChildOpen(false); }}>Add</Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    )
}