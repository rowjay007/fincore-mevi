"use client"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { History, ShieldCheck, FileText, AlertTriangle } from "lucide-react"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useEffect, useState } from "react"
import { faker } from "@faker-js/faker"

const generateAuditLog = () => ({
  id: faker.string.uuid().slice(0, 8),
  event: faker.helpers.arrayElement(["PAYMENT_SETTLED", "USER_LOGIN", "KEY_ROTATED", "ACCOUNT_CREATED"]),
  actor: faker.internet.username().toLowerCase(),
  metadata: "IP: " + faker.internet.ip(),
  hash: faker.git.commitSha().slice(0, 12) + "...",
  time: new Date().toLocaleTimeString(),
})

export default function AuditPage() {
  const [logs, setLogs] = useState<any[]>([])
  const [mounted, setMounted] = useState(false)

  useEffect(() => {
    setMounted(true)
    setLogs(Array.from({ length: 10 }, generateAuditLog))
  }, [])

  if (!mounted) return null

  return (
    <div className="space-y-8 max-w-7xl mx-auto">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Audit Ledger</h1>
          <p className="text-muted-foreground">Merkle Hash Chain for immutable event sequence verification.</p>
        </div>
        <div className="flex gap-2">
          <Badge variant="outline" className="bg-emerald-500/10 text-emerald-500 border-emerald-500/20 px-3 py-1">
            Sequence: VALID
          </Badge>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <Card className="bg-emerald-950/10 border-emerald-500/20">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Chain Integrity</CardTitle>
            <ShieldCheck className="h-4 w-4 text-emerald-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-emerald-500">TAMPER-PROOF</div>
            <p className="text-xs text-muted-foreground">Verified 2.4M blocks</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Event Count (24h)</CardTitle>
            <FileText className="h-4 w-4 text-blue-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">842,901</div>
            <p className="text-xs text-muted-foreground">+5.2% from yesterday</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Anomalies</CardTitle>
            <AlertTriangle className="h-4 w-4 text-emerald-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">0</div>
            <p className="text-xs text-muted-foreground">Last scan: 1 min ago</p>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Merkle-Linked Audit Stream</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Event ID</TableHead>
                <TableHead>Type</TableHead>
                <TableHead>Actor</TableHead>
                <TableHead>Merkle Proof (Prev Hash)</TableHead>
                <TableHead>Timestamp</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {logs.map((log) => (
                <TableRow key={log.id}>
                  <TableCell className="font-mono text-xs">{log.id}</TableCell>
                  <TableCell>
                    <Badge variant="secondary" className="font-mono text-[10px]">{log.event}</Badge>
                  </TableCell>
                  <TableCell className="text-xs">{log.actor}</TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">{log.hash}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">{log.time}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}
