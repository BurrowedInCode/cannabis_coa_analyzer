import { getMe } from '@/api/auth'
import { createFileRoute, Outlet, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/_authenticated')({
  component: RouteComponent,
  beforeLoad: async ({ context, location }) => {
    try {
      await context.queryClient.ensureQueryData({ queryKey: ['me'], queryFn: getMe, staleTime: 5 * 60 * 1000 })
    } catch {
      throw redirect({ to: '/login', search: { redirect: location.href } })
    }
  }
})

function RouteComponent() {
  return (<Outlet />)
}
