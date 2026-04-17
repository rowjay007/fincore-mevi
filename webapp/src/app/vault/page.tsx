"use client"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Lock, ShieldCheck, Key, Eye } from "lucide-react"
import { useEffect, useState } from "react"

export default function VaultPage() {
  const [mounted, setMounted] = useState(false)

  useEffect(() => {
    setMounted(true)
  }, [])

  if (!mounted) return null

  return (
    <div className="space-y-8 max-w-7xl mx-auto">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Security Vault</h1>
          <p className="text-muted-foreground">PCI-DSS tokenization and secret management via HashiCorp Vault.</p>
        </div>
        <Badge variant="outline" className="bg-emerald-500/10 text-emerald-500 border-emerald-500/20 px-3 py-1">
          Transit Engine: SEALED & LOKED
        </Badge>
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        <Card className="border-l-4 border-l-emerald-500">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <ShieldCheck className="h-5 w-5 text-emerald-500" />
              Cryptographic Policy
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="p-4 rounded-lg bg-muted/50 border space-y-2">
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">Encryption Algorithm</span>
                <span className="font-mono font-medium text-emerald-500">AES-256-GCM96</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">Key Rotation</span>
                <span className="font-medium text-emerald-500">Every 30 Days</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">HSM Integration</span>
                <span className="font-medium text-emerald-500">ACTIVE</span>
              </div>
            </div>
            <p className="text-sm text-muted-foreground">
              All PII data (PAN, IBAN, SSN) is tokenized at the gateway before hitting the application database.
            </p>
          </CardContent>
        </Card>

        <Card className="border-l-4 border-l-blue-500">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Lock className="h-5 w-5 text-blue-500" />
              Access Control (RBAC)
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="p-4 rounded-lg bg-muted/50 border space-y-2">
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">Active Roles</span>
                <span className="font-medium text-blue-500">4 Core / 2 Admin</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">4-Eyes Requirement</span>
                <span className="font-medium text-blue-500">ENABLED</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">Audit Traceability</span>
                <span className="font-medium text-blue-500">100% (Merkle-Linked)</span>
              </div>
            </div>
            <p className="text-sm text-muted-foreground">
              Administrative actions on sensitive keys require dual-authorization from independent security officers.
            </p>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Hardware Security Module (HSM) Status</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-4">
            {['US-East-1', 'EU-West-1', 'AP-South-1', 'Global-Failover'].map((region) => (
              <div key={region} className="p-4 border rounded-md text-center space-y-2">
                <div className="text-xs font-semibold text-muted-foreground uppercase">{region}</div>
                <div className="text-lg font-bold text-emerald-500">ONLINE</div>
                <Badge variant="outline" className="text-[10px]">99.999% SLA</Badge>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
