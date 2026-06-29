import { analyzeCOA } from '@/api/coa'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { useMutation } from '@tanstack/react-query'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useState } from 'react'

export const Route = createFileRoute('/upload')({
  component: Upload,
})

function Upload() {
  const navigate = useNavigate()
  const [file, setFile] = useState<File | null>(null)

  const mutation = useMutation({
    mutationFn: () => analyzeCOA(file),
    onSuccess: () => navigate({ to: "/analyses" }),
    onError: () => console.log("error sending"),
  })

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    mutation.mutate()
  }

  return (
    <div className="w-full px-4 py-4 flex justify-center">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>Upload COA</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} encType="multipart/form-data" className="space-y-4">
            <Input
              type="file"
              accept=".pdf"
              onChange={(e) => setFile(e.target.files?.[0] ?? null)}
            />
            <Button type="submit" className="w-full" disabled={!file || mutation.isPending}>
              {mutation.isPending ? "Analyzing..." : "Analyze"}
            </Button>
            {mutation.isError && (
              <p className="text-sm text-center text-destructive">Failed to analyze COA. Please try again.</p>
            )}
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
