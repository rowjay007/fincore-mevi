import * as React from "react"
import {
  LayoutDashboard,
  Wallet,
  ShieldAlert,
  History,
  Settings,
  Database,
  Globe,
  Lock,
} from "lucide-react"

import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
} from "@/components/ui/sidebar"

const data = {
  navMain: [
    {
      title: "Overview",
      url: "/",
      icon: LayoutDashboard,
      isActive: true,
    },
    {
      title: "Ledger",
      url: "/ledger",
      icon: Database,
    },
    {
      title: "Payments",
      url: "/payments",
      icon: Wallet,
    },
  ],
  modules: [
    {
      name: "Fraud Engine",
      url: "/fraud",
      icon: ShieldAlert,
    },
    {
      name: "Vault Service",
      url: "/vault",
      icon: Lock,
    },
    {
      name: "Audit Logs",
      url: "/audit",
      icon: History,
    },
    {
      name: "Geo-Regions",
      url: "/geo",
      icon: Globe,
    },
  ],
}

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton size="lg">
              <div className="flex aspect-square size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
                <Database className="size-4" />
              </div>
              <div className="flex flex-col gap-0.5 leading-none">
                <span className="font-semibold">FinCore</span>
                <span className="text-xs text-muted-foreground font-mono">v1.2.0</span>
              </div>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Main Navigation</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {data.navMain.map((item) => (
                <SidebarMenuItem key={item.title}>
                  <SidebarMenuButton asChild tooltip={item.title} isActive={item.isActive}>
                    <a href={item.url}>
                      <item.icon />
                      <span>{item.title}</span>
                    </a>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
        <SidebarGroup>
          <SidebarGroupLabel>Microservices</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {data.modules.map((item) => (
                <SidebarMenuItem key={item.name}>
                  <SidebarMenuButton asChild tooltip={item.name}>
                    <a href={item.url}>
                      <item.icon />
                      <span>{item.name}</span>
                    </a>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
      <SidebarRail />
    </Sidebar>
  )
}
