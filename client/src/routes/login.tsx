import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from "@/components/ui/card";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { UserLogin } from '@/api/auth';

export const Route = createFileRoute('/login')({
  component: Login,
})

function Login() {
  const navigate = useNavigate()

  const [username, setUsername] = useState<string>('')
  const [password, setPassword] = useState<string>('')

  const mutation = useMutation({
    mutationFn: () => UserLogin(username, password),
    onSuccess: () => { navigate({ to: '/upload' }) },
    onError: () => { console.log("error logging in") }
  })

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    mutation.mutate()
  }
  return (
    <div className="min-h-screen flex items-center justify-center bg-background">
      <div className="w-full max-w-sm px-4">
        <div className="text-center mb-8">
          <h1 className="text-2xl font-bold">Cannabis CoA Reader</h1>
          <p className="text-sm text-muted-foreground mt-1">Sign in to your account</p>
        </div>
        <form onSubmit={handleSubmit}>
          <Card>
            <CardContent className="pt-6">
              <FieldGroup>
                <Field>
                  <FieldLabel htmlFor="username">Username</FieldLabel>
                  <Input id="username" onChange={(e) => setUsername(e.target.value)} />
                </Field>
                <Field>
                  <FieldLabel htmlFor="password">Password</FieldLabel>
                  <Input id="password" type="password" onChange={(e) => setPassword(e.target.value)} />
                </Field>
              </FieldGroup>
              <Button type="submit" className="w-full mt-4" disabled={mutation.isPending}>
                {mutation.isPending ? "loading" : "sign in"}
              </Button>
            </CardContent>
          </Card>
        </form>
      </div>
    </div>
  )
}
