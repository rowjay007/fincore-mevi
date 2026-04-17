"use client"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Wallet, ArrowUpRight, ArrowDownLeft, Search, CheckCircle2, Clock } from "lucide-react"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useEffect, useState } from "react"
import { faker } from "@faker-js/faker"

const generatePayment = () => ({
  id: faker.string.uuid().slice(0, 8),
  sender: faker.finance.accountNumber(8),
  recipient: faker.finance.accountNumber(8),
  amount: faker.finance.amount({ min: 100, max: 10000 }),
  currency: faker.helpers.arrayElement(["USD", "EUR", "GBP"]),
  status: faker.helpers.arrayElement(["Settled", "Processing", "Authorized"]),
  time: new Date().toLocaleTimeString(),
})

export default function PaymentsPage() {
  const [payments, setPayments] = useState<any[]>([])
  const [mounted, setMounted] = useState(false)

  useEffect(() => {
    setMounted(true)
    setPayments(Array.from({ length: 8 }, generatePayment))
    
    const interval = setInterval(() => {
      setPayments(prev => [generatePayment(), ...prev.slice(0, 7)])
    }, 4000)
    return () => clearInterval(interval)
  }, [])

  if (!mounted) return null

  return (
    <div className="space-y-8 max-w-7xl mx-auto">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Payment Orchestration</h1>
          <p className="text-muted-foreground">Temporal-backed sagas for multi-service transactions.</p>
        </div>
        <Badge variant="outline" className="bg-blue-500/10 text-blue-500 border-blue-500/20 px-3 py-1">
          Worker Status: 12 Active
        </Badge>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Settled (24h)</CardTitle>
            <CheckCircle2 className="h-4 w-4 text-emerald-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">42,891</div>
            <p className="text-xs text-muted-foreground">Volume: $12.4M</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">In Flight</CardTitle>
            <Clock className="h-4 w-4 text-yellow-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">1,204</div>
            <p className="text-xs text-muted-foreground">Avg Latency: 420ms</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Volume</CardTitle>
            <Wallet className="h-4 w-4 text-blue-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">$142.8M</div>
            <p className="text-xs text-muted-foreground">Across all rails</p>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Global Payment Stream</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Trace ID</TableHead>
                <TableHead>Sender/Recipient</TableHead>
                <TableHead>Amount</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Time</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {payments.map((p) => (
                <TableRow key={p.id}>
                  <TableCell className="font-mono text-xs">{p.id}</TableCell>
                  <TableCell>
                    <div className="flex flex-col text-xs">
                      <span className="text-muted-foreground">From: {p.sender}</span>
                      <span className="font-medium">To: {p.recipient}</span>
                    </div>
                  </TableCell>
                  <TableCell className="font-semibold">
                    {p.amount} <span className="text-[10px] text-muted-foreground">{p.currency}</span>
                  </TableCell>
                  <TableCell>
                    <Badge variant="secondary" className={
                      p.status === "Settled" ? "bg-emerald-500/10 text-emerald-500 border-emerald-500/20" :
                      "bg-yellow-500/10 text-yellow-500 border-yellow-500/20"
                    }>
                      {p.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">{p.time}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}
