import { getCOAAnalysis } from '@/api/coa'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { useQuery } from '@tanstack/react-query'
import { createFileRoute, Link } from '@tanstack/react-router'

export const Route = createFileRoute('/analyses/$analysisID')({
  component: RouteComponent,
})

function TableSkeleton({ cols, rows = 4 }: { cols: number; rows?: number }) {
  return (
    <>
      {Array.from({ length: rows }).map((_, i) => (
        <TableRow key={i}>
          {Array.from({ length: cols }).map((_, j) => (
            <TableCell key={j}><Skeleton className="h-4 w-24" /></TableCell>
          ))}
        </TableRow>
      ))}
    </>
  )
}

const errorRow = (cols: number) => (
  <TableRow>
    <TableCell colSpan={cols} className="py-8 text-center text-muted-foreground">
      Failed to load data.
    </TableCell>
  </TableRow>
)

function RouteComponent() {
  const { analysisID } = Route.useParams()
  const { data: analysis, isLoading, isError } = useQuery({ queryKey: ['analysis', analysisID], queryFn: () => getCOAAnalysis(analysisID) })

  return (
    <div className="w-full px-4 py-4 space-y-6">
      <div>
        <Link to="/analyses" className="text-sm text-muted-foreground hover:text-foreground">
          ← Back to Analyses
        </Link>

        <h1 className=" mt-4 text-2xl font-semibold">Analysis: {analysis?.seed_to_sale_number ?? ""}</h1>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <Card>
          <CardHeader>
            <CardTitle>Analysis Info</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            {isError ? (
              <div className="flex justify-center text-muted-foreground">Failed to load analysis.</div>
            ) : (
              <>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Seed to Sale Number</span>
                  {isLoading ? <Skeleton className="h-4 w-32" /> : <span>{analysis?.seed_to_sale_number ?? ""}</span>}
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Sample Name</span>
                  {isLoading ? <Skeleton className="h-4 w-32" /> : <span>{analysis?.sample_name ?? ""}</span>}
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Test Date</span>
                  {isLoading ? <Skeleton className="h-4 w-32" /> : <span>{analysis?.test_date ?? ""}</span>}
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Sample Matrix</span>
                  {isLoading ? <Skeleton className="h-4 w-32" /> : <span>{analysis?.sample_matrix ?? ""}</span>}
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Overall Pass</span>
                  {isLoading ? <Skeleton className="h-4 w-32" /> : <span>{analysis?.overall_pass ? "Pass" : "Fail"}</span>}
                </div>
              </>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Laboratory</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            {isError ? (
              <div className="flex justify-center text-muted-foreground">Failed to load laboratory.</div>
            ) : (
              <>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Name</span>
                  {isLoading ? <Skeleton className="h-4 w-32" /> : <span>{analysis?.laboratory.name ?? ""}</span>}
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Address</span>
                  {isLoading ? <Skeleton className="h-4 w-32" /> : <span>{analysis?.laboratory.address ?? ""}</span>}
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Phone</span>
                  {isLoading ? <Skeleton className="h-4 w-32" /> : <span>{analysis?.laboratory.phone ?? ""}</span>}
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Certification</span>
                  {isLoading ? <Skeleton className="h-4 w-32" /> : <span>{analysis?.laboratory.certification ?? ""}</span>}
                </div>
              </>
            )}
          </CardContent>
        </Card>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <Card>
          <CardHeader>
            <CardTitle>Cannabinoids</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Value</TableHead>
                  <TableHead>Unit</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {isLoading ? <TableSkeleton cols={3} /> : isError ? errorRow(3) : analysis?.cannabinoids.map((c) =>
                  <TableRow key={c.name}>
                    <TableCell>{c.name}</TableCell>
                    <TableCell>{c.value}</TableCell>
                    <TableCell>{c.unit}</TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Terpenes</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Value</TableHead>
                  <TableHead>Unit</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {isLoading ? <TableSkeleton cols={3} /> : isError ? errorRow(3) : analysis?.terpenes.map((c) =>
                  <TableRow key={c.name}>
                    <TableCell>{c.name}</TableCell>
                    <TableCell>{c.value}</TableCell>
                    <TableCell>{c.unit}</TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Other Tests</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Status</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {isLoading ? <TableSkeleton cols={2} /> : isError ? errorRow(2) : analysis?.summary.map((c) =>
                  <TableRow key={c.name}>
                    <TableCell>{c.name}</TableCell>
                    <TableCell>{c.status}</TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
