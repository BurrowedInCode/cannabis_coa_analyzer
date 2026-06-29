import { getAllCOAAnalyses } from '@/api/coa'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useQuery } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'

export const Route = createFileRoute('/analyses/')({
  component: COATable,
})

function COATable() {
  const [page, setPage] = useState(1)
  const [limit] = useState(20)
  const offset = (page - 1) * limit

  const { data: analyses, isLoading, isError } = useQuery({ queryKey: ['analyses', limit, page], queryFn: () => getAllCOAAnalyses(limit, offset) })

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold">Analyses Summary</h1>
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Test Date</TableHead>
              <TableHead>Seed To Sale Number</TableHead>
              <TableHead>Sample Name</TableHead>
              <TableHead>Overall Pass</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              Array.from({ length: 4 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell><Skeleton className="h-4 w-10" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-32" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-32" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-10" /></TableCell>
                </TableRow>
              ))
            ) : isError ? (
              <TableRow>
                <TableCell colSpan={4} className="py-8 text-center text-muted-foreground">
                  Failed to load analyses.
                </TableCell>
              </TableRow>
            ) : analyses?.length === 0 ? (
              <TableRow>
                <TableCell colSpan={4} className="py-8 text-center text-muted-foreground">
                  No analyses found.
                </TableCell>
              </TableRow>
            ) : (
              analyses?.map((analysis) => (
                <TableRow key={analysis.id} className="cursor-pointer">
                  <TableCell>{
                    new Date(analysis.test_date).toLocaleDateString("en-US", {
                      month: "2-digit",
                      day: "2-digit",
                      year: "numeric",
                    }
                    )}
                  </TableCell>
                  <TableCell>{analysis.seed_to_sale_number}</TableCell>
                  <TableCell>{analysis.sample_name}</TableCell>
                  <TableCell>{analysis.overall_pass ? "Pass" : "Fail"}</TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      <div className="flex items-center justify-end gap-2">
        <Button
          variant="outline"
          size="sm"
          onClick={() => setPage(p => Math.max(1, p - 1))}
          disabled={page === 1 || isLoading}
        >
          Previous
        </Button>
        <span className="text-sm text-muted-foreground">Page {page}</span>
        <Button
          variant="outline"
          size="sm"
          onClick={() => setPage(p => p + 1)}
          disabled={(analyses?.length ?? 0) < limit || isLoading}
        >
          Next
        </Button>
      </div>
    </div>

  )
}
