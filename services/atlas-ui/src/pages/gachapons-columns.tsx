
import { type ColumnDef } from "@tanstack/react-table";
import type { GachaponData } from "@/types/models/gachapon";
import { Link } from "react-router-dom";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";

export const columns: ColumnDef<GachaponData>[] = [
  {
    accessorKey: "attributes.name",
    header: "Name",
    cell: ({ row }) => (
      <Link to={`/gachapons/${row.original.id}`} className="hover:underline">
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="font-medium">{row.original.attributes.name}</span>
            </TooltipTrigger>
            <TooltipContent copyable>
              <p>{row.original.id}</p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </Link>
    ),
  },
  {
    accessorKey: "attributes.commonWeight",
    header: "Common Weight",
  },
  {
    accessorKey: "attributes.uncommonWeight",
    header: "Uncommon Weight",
  },
  {
    accessorKey: "attributes.rareWeight",
    header: "Rare Weight",
  },
];
