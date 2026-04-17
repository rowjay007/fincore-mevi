"use client"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { ShieldAlert, Zap, Search, Fingerprint } from "lucide-react"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"

export default function FraudPage() {
  return (
    <div className="space-y-8 max-w-7xl mx-auto">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Fraud Detection Engine</h1>
          <p className="text-muted-foreground">Real-time risk evaluation and behavioral analysis.</p>
        </div>
        <Badge variant="outline" className="bg-emerald-500/10 text-emerald-500 border-emerald-500/20 px-3 py-1">
          Heuristic Evaluator: ACTIVE
        </Badge>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Scanned Today</CardTitle>
            <Zap className="h-4 w-4 text-yellow-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">1.2M</div>
            <p className="text-xs text-muted-foreground">Avg latency: 14ms</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Flagged Transactions</CardTitle>
            <ShieldAlert className="h-4 w-4 text-destructive" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">142</div>
            <p className="text-xs text-muted-foreground">0.01% of total volume</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Auto-Blocked</CardTitle>
            <Fingerprint className="h-4 w-4 text-blue-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">89</div>
            <p className="text-xs text-muted-foreground">Risk Score &gt; 0.90</p>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>High-Risk Queue (4-Eyes Approval)</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Tx ID</TableHead>
                <TableHead>Risk Score</TableHead>
                <TableHead>Reason</TableHead>
                <TableHead>Amount</TableHead>
                <TableHead>Action</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow>
                <TableCell className="font-mono">8a4082e6</TableCell>
                <TableCell>
                  <div className="flex items-center gap-2">
                    <div className="w-full bg-muted rounded-full h-2 max-w-[100px]">
                      <div className="bg-destructive h-2 rounded-full w-[85%]"></div>
                    </div>
                    <span className="text-xs font-bold text-destructive">0.85</span>
                  </div>
                </TableCell>
                <TableCell className="text-xs">Rapid velocity (5 tx/sec) from new device</TableCell>
                <TableCell className="font-medium">$12,000.00</TableCell>
                <TableCell>
                  <div className="flex gap-2">
                    <Badge className="cursor-pointer bg-emerald-600">APPROVE</Badge>
                    <Badge variant="destructive" className="cursor-pointer">REJECT</Badge>
                  </div>
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}
