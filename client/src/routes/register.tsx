import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from "@/components/ui/card";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { register } from '@/api/auth';

export const Route = createFileRoute('/register')({
  component: Register,
})

function Register() {
  const navigate = useNavigate()
  const [username, setUsername] = useState<string>("")
  const [password, setPassword] = useState<string>("")

  const mutation = useMutation({
    mutationFn: () => register(username, password),
    onSuccess: () => { navigate({ to: "/login" }) },
    onError: () => { console.log("error registering") }
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
          <p className="text-sm text-muted-foreground mt-1">Register</p>
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
                {mutation.isPending ? "loading" : "Register"}
              </Button>
              <p className="text-sm text-muted-foreground text-center mt-4">
                Already have an account?{" "}
                <Link to="/login" className="text-primary underline underline-offset-4">
                  Sign in
                </Link>
              </p>
            </CardContent>
          </Card>
        </form>
      </div>
    </div>
  )
}
