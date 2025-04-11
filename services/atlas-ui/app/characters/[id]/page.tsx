"use client"

import {useEffect, useState} from "react"
import {useParams} from "next/navigation"
import {Toaster} from "@/components/ui/sonner"
import {useTenant} from "@/context/tenant-context";
import {Card, CardContent, CardHeader, CardTitle} from "@/components/ui/card";
import {Character, fetchCharacter} from "@/lib/characters";

export default function GuildDetailPage() {
    const {id} = useParams()
    const {activeTenant} = useTenant()

    const [character, setCharacter] = useState<Character | null>(null)
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)

    useEffect(() => {
        if (!activeTenant || !id) return

        setLoading(true)
        fetchCharacter(activeTenant, String(id))
            .then(setCharacter)
            .catch((err) => setError(err.message))
            .finally(() => setLoading(false))
    }, [activeTenant, id])

    if (loading) return <div className="p-4">Loading...</div>
    if (error || !character) return <div className="p-4 text-red-500">Error: {error || "Character not found"}</div>

    return (
        <div className="flex flex-col flex-1 space-y-6 p-10 pb-16 h-screen">
            <div className="items-center justify-between space-y-2">
                <div>
                    <h2 className="text-2xl font-bold tracking-tight">{character.attributes.name}</h2>
                </div>
            </div>
            <div className="flex flex-row h-[25%] gap-6">
                <Card className="w-[100%]">
                    <CardHeader>
                        <CardTitle>Attributes</CardTitle>
                    </CardHeader>
                    <CardContent className="grid grid-cols-4 gap-2 text-sm text-muted-foreground">
                        <div>
                            <strong>World:</strong> {activeTenant?.attributes.worlds[character.attributes.worldId].name}
                        </div>
                        <div><strong>Gender:</strong> {character.attributes.gender}</div>
                        <div><strong>Level:</strong> {character.attributes.level}</div>
                        <div><strong>Experience:</strong> {character.attributes.experience}</div>
                        <div><strong>Strength:</strong> {character.attributes.strength}</div>
                        <div><strong>Dexterity:</strong> {character.attributes.dexterity}</div>
                        <div><strong>Intelligence:</strong> {character.attributes.intelligence}</div>
                        <div><strong>Luck:</strong> {character.attributes.luck}</div>
                    </CardContent>
                </Card>
            </div>
            <Toaster richColors/>
        </div>
    )
}
