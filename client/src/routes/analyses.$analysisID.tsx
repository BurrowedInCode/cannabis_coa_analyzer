import { getCOAAnalysis } from '@/api/coa'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import type { Analysis } from '@/types'
import { useQuery } from '@tanstack/react-query'
import { createFileRoute, Link } from '@tanstack/react-router'
import { useEffect, useState } from 'react'

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

  const [editing, setEditing] = useState(false)
  const [formData, setFormData] = useState<Analysis | null>(null)

  useEffect(() => {
    if (analysis) setFormData(analysis)
  }, [analysis])

  const handleCancel = () => {
    setFormData(analysis ?? null)
    setEditing(false)
  }

  const setField = (field: keyof Analysis, value: string | boolean) => {
    setFormData(prev => prev ? { ...prev, [field]: value } : prev)
  }

  const setLabField = (field: keyof Analysis['laboratory'], value: string) => {
    setFormData(prev => prev ? { ...prev, laboratory: { ...prev.laboratory, [field]: value } } : prev)
  }

  const setCannabinoidField = (i: number, field: 'name' | 'value' | 'unit', value: string) => {
    setFormData(prev => {
      if (!prev) return prev
      const cannabinoids = [...prev.cannabinoids]
      cannabinoids[i] = { ...cannabinoids[i], [field]: field === 'value' ? Number(value) : value }
      return { ...prev, cannabinoids }
    })
  }

  const setTerpeneField = (i: number, field: 'name' | 'value' | 'unit', value: string) => {
    setFormData(prev => {
      if (!prev) return prev
      const terpenes = [...prev.terpenes]
      terpenes[i] = { ...terpenes[i], [field]: field === 'value' ? Number(value) : value }
      return { ...prev, terpenes }
    })
  }

  const setSummaryField = (i: number, field: 'name' | 'status', value: string) => {
    setFormData(prev => {
      if (!prev) return prev
      const summary = [...prev.summary]
      summary[i] = { ...summary[i], [field]: value }
      return { ...prev, summary }
    })
  }

  return (
    <div className="w-full px-4 py-4 space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <Link to="/analyses" className="text-sm text-muted-foreground hover:text-foreground">
            ← Back to Analyses
          </Link>
          <h1 className="mt-4 text-2xl font-semibold">Analysis: {analysis?.seed_to_sale_number ?? ""}</h1>
        </div>
        <div className="flex gap-2 mt-4">
          {editing ? (
            <>
              <Button variant="outline" size="sm" onClick={handleCancel}>Cancel</Button>
              <Button size="sm" onClick={() => setEditing(false)}>Save</Button>
            </>
          ) : (
            <Button variant="outline" size="sm" onClick={() => setEditing(true)}>Edit</Button>
          )}
        </div>
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
                <div className="flex justify-between items-center">
                  <span className="text-muted-foreground">Seed to Sale Number</span>
                  {isLoading ? <Skeleton className="h-4 w-32" /> : editing ? (
                    <Input className="w-40 h-7 text-sm" value={formData?.seed_to_sale_number ?? ""} onChange={e => setField('seed_to_sale_number', e.target.value)} />
                  ) : <span>{formData?.seed_to_sale_number ?? ""}</span>}
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-muted-foreground">Sample Name</span>
                  {isLoading ? <Skeleton className="h-4 w-32" /> : editing ? (
                    <Input className="w-40 h-7 text-sm" value={formData?.sample_name ?? ""} onChange={e => setField('sample_name', e.target.value)} />
                  ) : <span>{formData?.sample_name ?? ""}</span>}
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-muted-foreground">Test Date</span>
                  {isLoading ? <Skeleton className="h-4 w-32" /> : editing ? (
                    <Input className="w-40 h-7 text-sm" value={formData?.test_date ?? ""} onChange={e => setField('test_date', e.target.value)} />
                  ) : <span>{formData?.test_date ?? ""}</span>}
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-muted-foreground">Sample Matrix</span>
                  {isLoading ? <Skeleton className="h-4 w-32" /> : editing ? (
                    <Input className="w-40 h-7 text-sm" value={formData?.sample_matrix ?? ""} onChange={e => setField('sample_matrix', e.target.value)} />
                  ) : <span>{formData?.sample_matrix ?? ""}</span>}
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-muted-foreground">Overall Pass</span>
                  {isLoading ? <Skeleton className="h-4 w-32" /> : <span>{formData?.overall_pass ? "Pass" : "Fail"}</span>}
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
                <div className="flex justify-between items-center">
                  <span className="text-muted-foreground">Name</span>
                  {isLoading ? <Skeleton className="h-4 w-32" /> : editing ? (
                    <Input className="w-40 h-7 text-sm" value={formData?.laboratory.name ?? ""} onChange={e => setLabField('name', e.target.value)} />
                  ) : <span>{formData?.laboratory.name ?? ""}</span>}
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-muted-foreground">Address</span>
                  {isLoading ? <Skeleton className="h-4 w-32" /> : editing ? (
                    <Input className="w-40 h-7 text-sm" value={formData?.laboratory.address ?? ""} onChange={e => setLabField('address', e.target.value)} />
                  ) : <span>{formData?.laboratory.address ?? ""}</span>}
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-muted-foreground">Phone</span>
                  {isLoading ? <Skeleton className="h-4 w-32" /> : editing ? (
                    <Input className="w-40 h-7 text-sm" value={formData?.laboratory.phone ?? ""} onChange={e => setLabField('phone', e.target.value)} />
                  ) : <span>{formData?.laboratory.phone ?? ""}</span>}
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-muted-foreground">Certification</span>
                  {isLoading ? <Skeleton className="h-4 w-32" /> : editing ? (
                    <Input className="w-40 h-7 text-sm" value={formData?.laboratory.certification ?? ""} onChange={e => setLabField('certification', e.target.value)} />
                  ) : <span>{formData?.laboratory.certification ?? ""}</span>}
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
                {isLoading ? <TableSkeleton cols={3} /> : isError ? errorRow(3) : formData?.cannabinoids.map((c, i) =>
                  <TableRow key={i}>
                    <TableCell>{editing ? <Input className="h-7 text-sm" value={c.name} onChange={e => setCannabinoidField(i, 'name', e.target.value)} /> : c.name}</TableCell>
                    <TableCell>{editing ? <Input className="h-7 text-sm" value={c.value} onChange={e => setCannabinoidField(i, 'value', e.target.value)} /> : c.value}</TableCell>
                    <TableCell>{editing ? <Input className="h-7 text-sm" value={c.unit} onChange={e => setCannabinoidField(i, 'unit', e.target.value)} /> : c.unit}</TableCell>
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
                {isLoading ? <TableSkeleton cols={3} /> : isError ? errorRow(3) : formData?.terpenes.map((t, i) =>
                  <TableRow key={i}>
                    <TableCell>{editing ? <Input className="h-7 text-sm" value={t.name} onChange={e => setTerpeneField(i, 'name', e.target.value)} /> : t.name}</TableCell>
                    <TableCell>{editing ? <Input className="h-7 text-sm" value={t.value} onChange={e => setTerpeneField(i, 'value', e.target.value)} /> : t.value}</TableCell>
                    <TableCell>{editing ? <Input className="h-7 text-sm" value={t.unit} onChange={e => setTerpeneField(i, 'unit', e.target.value)} /> : t.unit}</TableCell>
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
                {isLoading ? <TableSkeleton cols={2} /> : isError ? errorRow(2) : formData?.summary.map((s, i) =>
                  <TableRow key={i}>
                    <TableCell>{editing ? <Input className="h-7 text-sm" value={s.name} onChange={e => setSummaryField(i, 'name', e.target.value)} /> : s.name}</TableCell>
                    <TableCell>{editing ? <Input className="h-7 text-sm" value={s.status} onChange={e => setSummaryField(i, 'status', e.target.value)} /> : s.status}</TableCell>
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
