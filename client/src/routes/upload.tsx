import { analyzeCOA } from '@/api/coa'
import { Button } from '@/components/ui/button'
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
    onSuccess: () => { navigate({ to: "/table" }) },
    onError: () => { console.log("error sending") }
  })

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    mutation.mutate()
  }

  return (
    // Need to handle if user submits non pdf files
    <form onSubmit={handleSubmit} encType='multipart/form-data'>
      <Input type='file' accept='.pdf' onChange={(e) => setFile(e.target.files?.[0] ?? null)} />
      <Button type='submit'>{mutation.isPending ? "analyzing..." : "analyze"}</Button>
    </form>
  )
}
