import { createFileRoute } from '@tanstack/react-router'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from "@/components/ui/card";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";

export const Route = createFileRoute('/')({
  component: Index,
})

function Index() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-background">
      <div className="w-full max-w-sm px-4">
        <div className="text-center mb-8">
          <h1 className="text-2xl font-bold">Cannabis CoA Reader</h1>
          <p className="text-sm text-muted-foreground mt-1">Sign in to your account</p>
        </div>
        <form>
          <Card>
            <CardContent className="pt-6">
              <FieldGroup>
                <Field>
                  <FieldLabel htmlFor="username">Username</FieldLabel>
                  <Input id="username" autoFocus />
                </Field>
                <Field>
                  <FieldLabel htmlFor="password">Password</FieldLabel>
                  <Input id="password" type="password" autoComplete="current-password" />
                </Field>
              </FieldGroup>
              <Button type="submit" className="w-full mt-4" >
                Login
              </Button>
            </CardContent>
          </Card>
        </form>
      </div>
    </div>
  )
}
