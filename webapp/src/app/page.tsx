"use client"

import { useEffect, useState, useMemo } from "react"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import {
  Activity,
  ShieldCheck,
  Zap,
  AlertTriangle,
  ArrowUpRight,
  TrendingUp,
  Server,
  Globe,
  Cpu
} from "lucide-react"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { faker } from "@faker-js/faker"
import { toast } from "sonner"
import { motion, AnimatePresence } from "framer-motion"
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  Title,
  Tooltip as ChartTooltip,
  Legend,
  Filler,
} from 'chart.js'
import { Line, Bar } from 'react-chartjs-2'

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  Title,
  ChartTooltip,
  Legend,
  Filler
)

const generateTransaction = () => ({
  id: faker.string.uuid().slice(0, 12).toUpperCase(),
  amount: faker.finance.amount({ min: 100, max: 250000 }),
  currency: faker.helpers.arrayElement(["USD", "EUR", "GBP", "SGD", "JPY"]),
  type: faker.helpers.arrayElement(["PAYMENT", "TRANSFER", "FX_SWAP", "SETTLEMENT"]),
  status: faker.helpers.weightedArrayElement([
    { weight: 85, value: "COMPLETED" },
    { weight: 10, value: "PROCESSING" },
    { weight: 4, value: "PENDING" },
    { weight: 1, value: "FLAGGED" }
  ]),
  region: faker.helpers.arrayElement(["US-EAST-1", "EU-WEST-1", "AP-SOUTH-1"]),
  latency: faker.number.int({ min: 8, max: 45 }) + "ms",
  time: new Date().toLocaleTimeString('en-US', { hour12: false }),
})

export default function DashboardPage() {
  const [mounted, setMounted] = useState(false)
  const [transactions, setTransactions] = useState<any[]>([])
  const [metrics, setMetrics] = useState({
    tps: 10482,
    dailyVolume: "4.2B",
    activeNodes: 124,
    errorRate: "0.001%",
    merkleRoot: "8A4082E6...731F456"
  })
  const [tpsHistory, setTpsHistory] = useState([9200, 10100, 9800, 11200, 10500, 11800, 10900, 12100])

  const chartData = useMemo(() => ({
    labels: Array.from({ length: tpsHistory.length }, (_, i) => `T-${(tpsHistory.length - i) * 5}s`),
    datasets: [
      {
        fill: true,
        label: 'Real-time TPS',
        data: tpsHistory,
        borderColor: '#3b82f6',
        backgroundColor: 'rgba(59, 130, 246, 0.1)',
        borderWidth: 2,
        pointRadius: 0,
        tension: 0.4,
      },
    ],
  }), [tpsHistory])

  const regionalData = {
    labels: ['North America', 'Europe', 'Asia Pacific', 'LATAM'],
    datasets: [
      {
        label: 'Volume (Millions)',
        data: [4.2, 3.1, 2.4, 0.8],
        backgroundColor: 'rgba(59, 130, 246, 0.8)',
        borderRadius: 4,
      },
    ],
  }

  useEffect(() => {
    setMounted(true)
    setTransactions(Array.from({ length: 10 }, generateTransaction))

    const interval = setInterval(() => {
      const newTx = generateTransaction()
      setTransactions(prev => [newTx, ...prev.slice(0, 9)])

      const newTps = Math.floor(Math.random() * (12500 - 9500 + 1)) + 9500
      setMetrics(prev => ({
        ...prev,
        tps: newTps,
        merkleRoot: faker.git.commitSha().slice(0, 16).toUpperCase()
      }))
      setTpsHistory(prev => [...prev.slice(1), newTps])

      if (newTx.status === "FLAGGED") {
        toast.error(`SECURITY ALERT: HIGH RISK TRANSACTION`, {
          description: `Transaction ${newTx.id} triggered Heuristic Rule #42.`,
          duration: 10000,
        })
      }
    }, 3000)

    return () => clearInterval(interval)
  }, [])

  if (!mounted) return null

  return (
    <div className="space-y-6 max-w-[1600px] mx-auto pb-10">
      {/* Dynamic Header */}
      <div className="flex flex-col lg:flex-row justify-between items-start lg:items-center gap-4 bg-card/50 p-6 rounded-xl border border-border/50">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-3xl font-extrabold tracking-tight">FinCore Command Center</h1>
            <Badge className="bg-emerald-500/10 text-emerald-500 border-emerald-500/20 animate-pulse font-mono">LIVE PRODUCTION</Badge>
          </div>
          <p className="text-muted-foreground mt-1 font-medium">Global Liquidity Monitoring • 12 Active Services</p>
        </div>
        <div className="flex flex-wrap items-center gap-4 bg-background/50 p-2 rounded-lg border">
          <div className="flex items-center gap-2 px-3 border-r">
            <Server className="h-4 w-4 text-blue-400" />
            <span className="text-xs font-mono">{metrics.activeNodes} Nodes Online</span>
          </div>
          <div className="flex items-center gap-2 px-3 border-r">
            <Globe className="h-4 w-4 text-emerald-400" />
            <span className="text-xs font-mono">Quorum: REACHED</span>
          </div>
          <div className="flex items-center gap-2 px-3">
            <Cpu className="h-4 w-4 text-purple-400" />
            <span className="text-xs font-mono">Err: {metrics.errorRate}</span>
          </div>
        </div>
      </div>

      {/* KPI Grid */}
      <div className="grid gap-4 grid-cols-1 md:grid-cols-2 lg:grid-cols-4">
        <Card className="bg-gradient-to-br from-card to-background border-l-4 border-l-blue-500 shadow-xl overflow-hidden relative">
          <div className="absolute top-0 right-0 p-2 opacity-5">
            <Zap className="h-24 w-24" />
          </div>
          <CardHeader className="pb-2">
            <CardTitle className="text-xs font-bold uppercase tracking-widest text-muted-foreground">Throughput (TPS)</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-black font-mono">{metrics.tps.toLocaleString()}</div>
            <div className="flex items-center gap-1 mt-2 text-xs font-bold text-emerald-500">
              <TrendingUp className="h-3 w-3" />
              <span>+4.2% Peak Velocity</span>
            </div>
          </CardContent>
        </Card>

        <Card className="bg-gradient-to-br from-card to-background border-l-4 border-l-emerald-500 shadow-xl">
          <CardHeader className="pb-2">
            <CardTitle className="text-xs font-bold uppercase tracking-widest text-muted-foreground">Daily Volume (USD)</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-black font-mono">${metrics.dailyVolume}</div>
            <div className="flex items-center gap-1 mt-2 text-xs font-medium text-muted-foreground">
              <span>Settlement Window: 18h remaining</span>
            </div>
          </CardContent>
        </Card>

        <Card className="bg-gradient-to-br from-card to-background border-l-4 border-l-purple-500 shadow-xl">
          <CardHeader className="pb-2">
            <CardTitle className="text-xs font-bold uppercase tracking-widest text-muted-foreground">Active Sagas</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-black font-mono">184,202</div>
            <div className="flex items-center gap-1 mt-2 text-xs font-bold text-blue-500">
              <Activity className="h-3 w-3 animate-spin" style={{ animationDuration: '3s' }} />
              <span>Temporal Managed</span>
            </div>
          </CardContent>
        </Card>

        <Card className="bg-gradient-to-br from-card to-background border-l-4 border-l-red-500 shadow-xl">
          <CardHeader className="pb-2">
            <CardTitle className="text-xs font-bold uppercase tracking-widest text-muted-foreground">Audit Root Hash</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-sm font-black font-mono text-destructive truncate">{metrics.merkleRoot}</div>
            <div className="flex items-center gap-1 mt-2 text-xs font-bold text-red-500">
              <ShieldCheck className="h-3 w-3" />
              <span>Immutable Chain Verified</span>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Analytics Row */}
      <div className="grid gap-6 grid-cols-1 lg:grid-cols-12">
        <Card className="lg:col-span-8 shadow-2xl border-border/50">
          <CardHeader className="flex flex-row items-center justify-between">
            <div>
              <CardTitle>System Load & Network Pressure</CardTitle>
              <CardDescription>Real-time telemetry across geo-distributed cluster.</CardDescription>
            </div>
            <div className="flex gap-2">
              <Badge variant="outline" className="bg-blue-500/5">Load: Normal</Badge>
              <Badge variant="outline" className="bg-emerald-500/5">Health: 100%</Badge>
            </div>
          </CardHeader>
          <CardContent>
            <div className="h-[400px] w-full">
              <Line
                data={chartData}
                options={{
                  responsive: true,
                  maintainAspectRatio: false,
                  animation: { duration: 800 },
                  scales: {
                    y: { grid: { color: 'rgba(255,255,255,0.05)' }, ticks: { color: '#666', font: { family: 'monospace' } } },
                    x: { grid: { display: false }, ticks: { color: '#666', font: { family: 'monospace' } } }
                  },
                  plugins: { legend: { display: false } }
                } as any}
              />
            </div>
          </CardContent>
        </Card>

        <Card className="lg:col-span-4 shadow-2xl border-border/50">
          <CardHeader>
            <CardTitle>Regional Volume (24h)</CardTitle>
            <CardDescription>Liquidity distribution by region.</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="h-[300px] w-full mt-4">
              <Bar
                data={regionalData}
                options={{
                  responsive: true,
                  maintainAspectRatio: false,
                  plugins: { legend: { display: false } },
                  scales: {
                    y: { grid: { display: false }, ticks: { display: false } },
                    x: { grid: { display: false }, ticks: { color: '#888', font: { size: 10 } } }
                  }
                }}
              />
            </div>
            <div className="mt-6 space-y-4">
              {regionalData.labels.map((label, i) => (
                <div key={label} className="flex justify-between items-center text-sm">
                  <span className="text-muted-foreground font-medium">{label}</span>
                  <div className="flex items-center gap-3">
                    <span className="font-mono font-bold">${regionalData.datasets[0].data[i]}B</span>
                    <Badge className="bg-emerald-500/10 text-emerald-500 border-none text-[10px]">+2.1%</Badge>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Real-time Ingestion Engine */}
      <Card className="shadow-2xl border-border/50 overflow-hidden">
        <CardHeader className="bg-muted/30 border-b flex flex-row items-center justify-between py-4 px-6">
          <div>
            <CardTitle className="text-lg">Kafka Ingestion Engine</CardTitle>
            <CardDescription>Live outbox relay • Zero-trust verified</CardDescription>
          </div>
          <div className="flex items-center gap-2 px-3 py-1 bg-background rounded-full border text-[10px] font-mono shadow-inner">
            <div className="h-2 w-2 bg-emerald-500 rounded-full animate-ping" />
            INGESTING 842.1 MB/s
          </div>
        </CardHeader>
        <CardContent className="p-0">
          <div className="overflow-x-auto">
            <Table>
              <TableHeader className="bg-muted/10">
                <TableRow className="border-none">
                  <TableHead className="w-[180px] pl-6 uppercase text-[10px] font-black tracking-tighter">Trace ID</TableHead>
                  <TableHead className="uppercase text-[10px] font-black tracking-tighter text-center">Type</TableHead>
                  <TableHead className="uppercase text-[10px] font-black tracking-tighter">Region</TableHead>
                  <TableHead className="uppercase text-[10px] font-black tracking-tighter">Latency</TableHead>
                  <TableHead className="uppercase text-[10px] font-black tracking-tighter text-right">Amount</TableHead>
                  <TableHead className="uppercase text-[10px] font-black tracking-tighter text-center pr-6">State</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <AnimatePresence mode="popLayout">
                  {transactions.map((tx) => (
                    <motion.tr
                      layout
                      initial={{ opacity: 0, x: -20 }}
                      animate={{ opacity: 1, x: 0 }}
                      exit={{ opacity: 0, x: 20 }}
                      key={tx.id}
                      className="group transition-all hover:bg-blue-500/5 border-b border-border/30 last:border-0"
                    >
                      <TableCell className="font-mono text-[11px] font-bold pl-6 text-blue-400 py-4">
                        {tx.id}
                      </TableCell>
                      <TableCell className="text-center">
                        <span className="text-[10px] font-black bg-muted px-2 py-1 rounded text-muted-foreground group-hover:text-foreground group-hover:bg-primary/20 transition-colors">
                          {tx.type}
                        </span>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2 text-xs font-medium">
                          <Globe className="h-3 w-3 text-muted-foreground" />
                          {tx.region}
                        </div>
                      </TableCell>
                      <TableCell className="font-mono text-xs text-muted-foreground">
                        {tx.latency}
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex flex-col">
                          <span className="font-black text-sm">
                            {new Intl.NumberFormat('en-US', { style: 'currency', currency: tx.currency }).format(tx.amount)}
                          </span>
                          <span className="text-[9px] text-muted-foreground font-mono">{tx.time}</span>
                        </div>
                      </TableCell>
                      <TableCell className="text-center pr-6">
                        <Badge
                          variant="secondary"
                          className={
                            tx.status === "FLAGGED" ? "bg-red-500/10 text-red-500 border-red-500/20" :
                              tx.status === "PENDING" ? "bg-yellow-500/10 text-yellow-500 border-yellow-500/20" :
                                tx.status === "PROCESSING" ? "bg-blue-500/10 text-blue-500 border-blue-500/20" :
                                  "bg-emerald-500/10 text-emerald-500 border-emerald-500/20"
                          }
                        >
                          <div className={`mr-1.5 h-1.5 w-1.5 rounded-full ${tx.status === "FLAGGED" ? "bg-red-500 shadow-[0_0_8px_rgba(239,68,68,0.8)]" :
                            tx.status === "COMPLETED" ? "bg-emerald-500" :
                              "bg-current animate-pulse"
                            }`} />
                          {tx.status}
                        </Badge>
                      </TableCell>
                    </motion.tr>
                  ))}
                </AnimatePresence>
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
