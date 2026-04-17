"use client"

import { useEffect, useState } from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Activity, ShieldCheck, Zap, AlertTriangle, ArrowUpRight, ArrowDownLeft } from "lucide-react"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { faker } from "@faker-js/faker"
import { toast } from "sonner"
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip as ChartTooltip,
  Legend,
  Filler,
} from 'chart.js'
import { Line } from 'react-chartjs-2'

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  ChartTooltip,
  Legend,
  Filler
)

// Mock data generator for live feeling
const generateTransaction = () => ({
  id: faker.string.uuid().slice(0, 8),
  amount: faker.finance.amount({ min: 10, max: 50000 }),
  currency: faker.helpers.arrayElement(["USD", "EUR", "GBP", "JPY"]),
  type: faker.helpers.arrayElement(["Credit", "Debit"]),
  status: faker.helpers.arrayElement(["Completed", "Pending", "Flagged"]),
  time: new Date().toLocaleTimeString(),
})

export default function DashboardPage() {
  const [mounted, setMounted] = useState(false)
  const [transactions, setTransactions] = useState<any[]>([])
  const [tps, setTps] = useState(8400)

  const chartData = {
    labels: ["10:00", "10:05", "10:10", "10:15", "10:20", "10:25"],
    datasets: [
      {
        fill: true,
        label: 'Throughput (TPS)',
        data: [4500, 7200, 9800, 10400, 8900, 11000],
        borderColor: 'rgb(59, 130, 246)',
        backgroundColor: 'rgba(59, 130, 246, 0.1)',
        tension: 0.4,
      },
    ],
  }

  const chartOptions = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: {
        display: false,
      },
      tooltip: {
        mode: 'index' as const,
        intersect: false,
      },
    },
    scales: {
      y: {
        grid: {
          display: false,
        },
        ticks: {
          color: '#888888',
        },
      },
      x: {
        grid: {
          display: false,
        },
        ticks: {
          color: '#888888',
        },
      },
    },
  }

  // Handle hydration
  useEffect(() => {
    setMounted(true)
    const initialTransactions = Array.from({ length: 6 }, generateTransaction)
    setTransactions(initialTransactions)

    const interval = setInterval(() => {
      const newTx = generateTransaction()

      setTransactions(prev => [newTx, ...prev.slice(0, 5)])

      const newTps = Math.floor(Math.random() * (12000 - 8000 + 1)) + 8000
      setTps(newTps)

      if (newTx.status === "Flagged") {
        toast.error(`High Risk Detected: Tx ${newTx.id}`, {
          description: `Suspicious ${newTx.amount} ${newTx.currency} movement detected.`,
        })
      }
    }, 5000)

    return () => clearInterval(interval)
  }, [])

  if (!mounted) return null

  return (
    <div className="space-y-8 max-w-7xl mx-auto">
      {/* Header */}
      <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">FinCore Master Dashboard</h1>
          <p className="text-muted-foreground">Real-time status of your 12-service banking ecosystem.</p>
        </div>
        <div className="flex items-center gap-2">
          <Badge variant="outline" className="bg-emerald-500/10 text-emerald-500 border-emerald-500/20 px-3 py-1">
            System Healthy
          </Badge>
          <Badge variant="outline" className="bg-blue-500/10 text-blue-500 border-blue-500/20 px-3 py-1">
            Geo-Scale: ACTIVE
          </Badge>
        </div>
      </div>

      {/* Stats Grid */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card className="border-l-4 border-l-yellow-500 shadow-sm">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Throughput (TPS)</CardTitle>
            <Zap className="h-4 w-4 text-yellow-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold tracking-tight">{tps.toLocaleString()}</div>
            <p className="text-xs text-muted-foreground flex items-center gap-1 mt-1">
              <ArrowUpRight className="h-3 w-3 text-emerald-500" /> +12% from last hour
            </p>
          </CardContent>
        </Card>
        <Card className="border-l-4 border-l-emerald-500 shadow-sm">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Fraud Score (Avg)</CardTitle>
            <ShieldCheck className="h-4 w-4 text-emerald-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold tracking-tight">0.03</div>
            <p className="text-xs text-muted-foreground mt-1 text-emerald-500 font-medium">Risk: Low</p>
          </CardContent>
        </Card>
        <Card className="border-l-4 border-l-blue-500 shadow-sm">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Active Sagas</CardTitle>
            <Activity className="h-4 w-4 text-blue-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold tracking-tight">1,204</div>
            <p className="text-xs text-muted-foreground mt-1">Orchestrated by Temporal</p>
          </CardContent>
        </Card>
        <Card className="border-l-4 border-l-destructive shadow-sm">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Fraud Alerts</CardTitle>
            <AlertTriangle className="h-4 w-4 text-destructive" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-destructive tracking-tight">2</div>
            <p className="text-xs text-muted-foreground mt-1">Requiring 4-eyes approval</p>
          </CardContent>
        </Card>
      </div>

      {/* Main Content Area */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-7">
        <Card className="col-span-4 shadow-sm">
          <CardHeader>
            <CardTitle>System Load (TPS)</CardTitle>
          </CardHeader>
          <CardContent className="pl-2">
            <div className="h-[350px] w-full">
              <Line data={chartData} options={chartOptions} />
            </div>
          </CardContent>
        </Card>

        <Card className="col-span-3 shadow-sm">
          <CardHeader>
            <CardTitle>Audit Integrity Vault</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-6">
              <div className="flex items-center gap-4 rounded-lg border p-4 bg-muted/30">
                <ShieldCheck className="h-5 w-5 text-emerald-500" />
                <div className="flex-1 space-y-1">
                  <p className="text-sm font-medium leading-none">Merkle Chain Validated</p>
                  <p className="text-xs text-muted-foreground">Last verified: 2 minutes ago</p>
                </div>
                <Badge variant="secondary" className="font-mono text-[10px]">LOCKED</Badge>
              </div>
              <div className="space-y-2">
                <label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Merkle Root Hash</label>
                <div className="text-xs font-mono bg-muted p-3 rounded-md border border-dashed border-muted-foreground/30 break-all leading-relaxed">
                  8a4082e61731f456c6b8a21f37e8c1...
                </div>
              </div>
              <p className="text-sm text-muted-foreground leading-relaxed">
                The audit-service has mathematically proven the integrity of all <span className="text-foreground font-semibold">2.4M</span> transactions stored in CockroachDB across 3 geo-regions.
              </p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Transaction Feed */}
      <Card className="shadow-sm">
        <CardHeader className="flex flex-row items-center justify-between py-4">
          <CardTitle>Live Transaction Feed</CardTitle>
          <Badge variant="outline" className="font-mono text-[10px] animate-pulse border-emerald-500/50 text-emerald-500">LIVE INGESTION</Badge>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader className="bg-muted/30">
              <TableRow>
                <TableHead className="w-[100px] pl-6">Tx ID</TableHead>
                <TableHead>Time</TableHead>
                <TableHead>Amount</TableHead>
                <TableHead>Type</TableHead>
                <TableHead className="pr-6">Status</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {transactions.map((tx) => (
                <TableRow key={tx.id} className="group transition-colors hover:bg-muted/50 border-b last:border-0">
                  <TableCell className="font-mono text-xs pl-6">{tx.id}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">{tx.time}</TableCell>
                  <TableCell className="font-medium">
                    {tx.amount} <span className="text-[10px] text-muted-foreground font-normal">{tx.currency}</span>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      {tx.type === "Credit" ? (
                        <ArrowDownLeft className="h-3 w-3 text-emerald-500" />
                      ) : (
                        <ArrowUpRight className="h-3 w-3 text-blue-500" />
                      )}
                      <span className="text-xs">{tx.type}</span>
                    </div>
                  </TableCell>
                  <TableCell className="pr-6">
                    <Badge
                      variant="secondary"
                      className={
                        tx.status === "Flagged" ? "bg-destructive/10 text-destructive border-destructive/20" :
                          tx.status === "Pending" ? "bg-yellow-500/10 text-yellow-500 border-yellow-500/20" :
                            "bg-emerald-500/10 text-emerald-500 border-emerald-500/20"
                      }
                    >
                      {tx.status}
                    </Badge>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}
