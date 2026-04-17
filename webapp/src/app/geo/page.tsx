"use client"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Globe, MapPin, Server, Activity } from "lucide-react"
import { useEffect, useState } from "react"

const regions = [
  { name: "US-East-1 (Northern Virginia)", status: "Active", latency: "12ms", load: "34%" },
  { name: "EU-West-1 (Ireland)", status: "Active", latency: "24ms", load: "28%" },
  { name: "AP-South-1 (Mumbai)", status: "Active", latency: "18ms", load: "19%" },
]

export default function GeoPage() {
  const [mounted, setMounted] = useState(false)

  useEffect(() => {
    setMounted(true)
  }, [])

  if (!mounted) return null

  return (
    <div className="space-y-8 max-w-7xl mx-auto">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Geo-Partitioned Cluster</h1>
          <p className="text-muted-foreground">Multi-region CockroachDB and Service deployment for data sovereignty.</p>
        </div>
        <Badge variant="outline" className="bg-blue-500/10 text-blue-500 border-blue-500/20 px-3 py-1">
          Global Consensus: REACHED
        </Badge>
      </div>

      <div className="grid gap-6 md:grid-cols-3">
        {regions.map((r) => (
          <Card key={r.name} className="border-l-4 border-l-blue-500">
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">{r.name}</CardTitle>
              <MapPin className="h-4 w-4 text-blue-500" />
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex justify-between items-center">
                <span className="text-xs text-muted-foreground uppercase font-semibold tracking-wider">Status</span>
                <Badge className="bg-emerald-600/10 text-emerald-500 border-none h-5 text-[10px] uppercase">{r.status}</Badge>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-xs text-muted-foreground uppercase font-semibold tracking-wider">P99 Latency</span>
                <span className="text-sm font-mono">{r.latency}</span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-xs text-muted-foreground uppercase font-semibold tracking-wider">Compute Load</span>
                <span className="text-sm font-mono">{r.load}</span>
              </div>
              <div className="w-full bg-muted rounded-full h-1.5 mt-2">
                <div className="bg-blue-500 h-1.5 rounded-full" style={{ width: r.load }}></div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Globe className="h-5 w-5 text-blue-500" />
            Global Compliance & Sovereignty
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            <div className="p-4 rounded-lg bg-muted/50 border space-y-2">
              <h3 className="font-semibold text-sm">GDPR Compliance (EU)</h3>
              <p className="text-xs text-muted-foreground leading-relaxed">
                European citizen data is pinned to <code>EU-West-1</code> using CockroachDB regional-by-row partitioning.
                Secrets never leave the region.
              </p>
              <Badge variant="outline" className="text-[10px] text-emerald-500 border-emerald-500/20">PINNED</Badge>
            </div>
            <div className="p-4 rounded-lg bg-muted/50 border space-y-2">
              <h3 className="font-semibold text-sm">Local Survivor Goal</h3>
              <p className="text-xs text-muted-foreground leading-relaxed">
                The cluster is configured to survive a complete region failure (AZ-level) without data loss, 
                maintaining ACID consistency.
              </p>
              <Badge variant="outline" className="text-[10px] text-blue-500 border-blue-500/20">CONFIGURED</Badge>
            </div>
          </div>
        </CardContent>
      </Card>

      <div className="p-4 rounded-lg border border-dashed border-muted-foreground/30 flex items-center justify-center gap-4 text-sm text-muted-foreground">
        <Server className="h-4 w-4" />
        Synchronized with Terraform Cloud & ArgoCD for Global Rollouts
        <Activity className="h-3 w-3 text-emerald-500 animate-pulse" />
      </div>
    </div>
  )
}
