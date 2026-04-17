"use client"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Database, ShieldCheck, History, Search } from "lucide-react"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"

export default function LedgerPage() {
  return (
    <div className="space-y-8 max-w-7xl mx-auto">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Triple-Entry Ledger</h1>
          <p className="text-muted-foreground">Immutable financial record of all asset movements.</p>
        </div>
        <Badge variant="outline" className="bg-emerald-500/10 text-emerald-500 border-emerald-500/20 px-3 py-1">
          Synced with CockroachDB
        </Badge>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Assets</CardTitle>
            <Database className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">$1.24B</div>
            <p className="text-xs text-muted-foreground">Across 3 regions</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Integrity Status</CardTitle>
            <ShieldCheck className="h-4 w-4 text-emerald-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">100.00%</div>
            <p className="text-xs text-muted-foreground">Zero variance detected</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Last Entry</CardTitle>
            <History className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">2ms ago</div>
            <p className="text-xs text-muted-foreground">ID: 8a4082e6</p>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <div className="flex justify-between items-center">
            <CardTitle>Recent Ledger Entries</CardTitle>
            <div className="flex items-center gap-2 border rounded-md px-3 py-1 bg-muted/50">
              <Search className="h-4 w-4 text-muted-foreground" />
              <input 
                placeholder="Search Account ID..." 
                className="bg-transparent border-none text-sm focus:outline-none w-64"
              />
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Account ID</TableHead>
                <TableHead>Debit</TableHead>
                <TableHead>Credit</TableHead>
                <TableHead>Balance</TableHead>
                <TableHead>Currency</TableHead>
                <TableHead>Timestamp</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow>
                <TableCell className="font-mono">ACC-8291</TableCell>
                <TableCell className="text-destructive">-500.00</TableCell>
                <TableCell>0.00</TableCell>
                <TableCell>12,450.00</TableCell>
                <TableCell>USD</TableCell>
                <TableCell>12:45:01</TableCell>
              </TableRow>
              <TableRow>
                <TableCell className="font-mono">ACC-4512</TableCell>
                <TableCell>0.00</TableCell>
                <TableCell className="text-emerald-500">+500.00</TableCell>
                <TableCell>8,902.50</TableCell>
                <TableCell>USD</TableCell>
                <TableCell>12:45:01</TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}
