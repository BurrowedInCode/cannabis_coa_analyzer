import { logout } from "@/api/auth";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate } from "@tanstack/react-router";
import { Button } from '@/components/ui/button'


export function NavBar() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()

  const mutation = useMutation({
    mutationFn: logout,
    onSuccess: () => {
      queryClient.removeQueries({ queryKey: ['me'] })
      navigate({ to: "/login" })
    }
  })
  const linkClass = "text-sm text-muted-foreground transition-colors hover:text-foreground"
  const activeClass = "text-foreground font-medium"

  return (
    <header className="border-b bg-background">
      <nav className="mx-auto flex h-14 max-w-5xl items-center justify-between px-4">
        <Link to="/upload" className="font-semibold">
          Cannabis CoA Reader
        </Link>
        <div className="flex items-center gap-6">
          <Link to="/upload" className={linkClass} activeProps={{ className: activeClass }}>
            Upload
          </Link>
          <Link to="/analyses" className={linkClass} activeProps={{ className: activeClass }}>
            Analyses
          </Link>
          <Button
            variant="outline"
            size="sm"
            onClick={() => mutation.mutate()}
            disabled={mutation.isPending}
          >
            {mutation.isPending ? "..." : "Logout"}
          </Button>
        </div>
      </nav>
    </header>
  )
}
