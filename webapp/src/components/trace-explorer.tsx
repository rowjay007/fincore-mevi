"use client"

import React, { useState, useEffect } from "react"
import { motion, AnimatePresence } from "framer-motion"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { CheckCircle2, AlertTriangle, ArrowRight, Server, Database, ShieldCheck, Zap } from "lucide-react"

interface TraceSpan {
  id: string
  service: string
  operation: string
  duration: number
  status: "success" | "error" | "pending"
  timestamp: string
}

interface Trace {
  id: string
  startTime: string
  totalDuration: number
  spans: TraceSpan[]
}

const services = [
  { name: "api-gateway", icon: Server, color: "text-blue-500", bg: "bg-blue-500/10" },
  { name: "auth-service", icon: ShieldCheck, color: "text-purple-500", bg: "bg-purple-500/10" },
  { name: "payment-service", icon: Zap, color: "text-yellow-500", bg: "bg-yellow-500/10" },
  { name: "ledger-service", icon: Database, color: "text-emerald-500", bg: "bg-emerald-500/10" },
  { name: "fraud-engine", icon: AlertTriangle, color: "text-red-500", bg: "bg-red-500/10" }
]

const mockTraces: Trace[] = [
  {
    id: "tr-9b2c-8821",
    startTime: "11:05:22",
    totalDuration: 142,
    spans: [
      { id: "s1", service: "api-gateway", operation: "POST /v1/payments", duration: 12, status: "success", timestamp: "11:05:22.010" },
      { id: "s2", service: "auth-service", operation: "ValidateToken", duration: 8, status: "success", timestamp: "11:05:22.022" },
      { id: "s3", service: "fraud-engine", operation: "EvaluateRisk", duration: 45, status: "success", timestamp: "11:05:22.030" },
      { id: "s4", service: "payment-service", operation: "InitiatePayment", duration: 32, status: "success", timestamp: "11:05:22.075" },
      { id: "s5", service: "ledger-service", operation: "RecordEntry", duration: 45, status: "success", timestamp: "11:05:22.107" }
    ]
  }
]

export function TraceExplorer() {
  const [selectedTrace, setSelectedTrace] = useState<Trace | null>(mockTraces[0])

  return (
    <Card className="shadow-2xl border-border/50 overflow-hidden bg-card/30 backdrop-blur-xl">
      <CardHeader className="border-b bg-muted/20">
        <div className="flex justify-between items-center">
          <div>
            <CardTitle className="text-xl font-bold font-mono">DISTRIBUTED_TRACE_EXPLORER</CardTitle>
            <CardDescription>Visualizing OpenTelemetry spans across 12 services</CardDescription>
          </div>
          <Badge variant="outline" className="font-mono text-[10px] animate-pulse bg-blue-500/10 text-blue-400">
            INGESTING_SPANS
          </Badge>
        </div>
      </CardHeader>
      <CardContent className="p-0">
        <div className="grid grid-cols-1 lg:grid-cols-12 min-h-[500px]">
          {/* Trace List */}
          <div className="lg:col-span-4 border-r border-border/30 overflow-y-auto max-h-[600px] p-4 space-y-3 bg-muted/5">
            <div className="text-[10px] font-black uppercase text-muted-foreground tracking-widest mb-4">Latest Traces</div>
            {mockTraces.map((trace) => (
              <div
                key={trace.id}
                onClick={() => setSelectedTrace(trace)}
                className={`p-4 rounded-lg border cursor-pointer transition-all ${
                  selectedTrace?.id === trace.id 
                  ? "bg-blue-500/10 border-blue-500/50 shadow-[0_0_15px_rgba(59,130,246,0.1)]" 
                  : "bg-background/50 border-border/50 hover:border-muted-foreground/50"
                }`}
              >
                <div className="flex justify-between items-start mb-2">
                  <span className="font-mono text-xs font-bold text-blue-400">{trace.id}</span>
                  <span className="text-[10px] text-muted-foreground">{trace.startTime}</span>
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-lg font-black">{trace.totalDuration}ms</span>
                  <Badge className="text-[9px] bg-emerald-500/10 text-emerald-400 border-none">HEALTHY</Badge>
                </div>
              </div>
            ))}
          </div>

          {/* Trace Visualization */}
          <div className="lg:col-span-8 p-6 bg-background/20 relative overflow-hidden">
            <div className="absolute inset-0 opacity-5 pointer-events-none" 
                 style={{ backgroundImage: 'radial-gradient(#3b82f6 1px, transparent 1px)', backgroundSize: '20px 20px' }} />
            
            <AnimatePresence mode="wait">
              {selectedTrace && (
                <motion.div
                  initial={{ opacity: 0, y: 20 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -20 }}
                  className="space-y-8"
                >
                  <div className="flex items-center justify-between">
                    <h3 className="font-mono font-black text-sm uppercase tracking-tighter">Execution Flow Architecture</h3>
                    <div className="flex gap-2">
                      <div className="flex items-center gap-1 text-[10px] font-bold text-muted-foreground">
                        <div className="h-1.5 w-1.5 rounded-full bg-blue-500" /> SYNC
                      </div>
                      <div className="flex items-center gap-1 text-[10px] font-bold text-muted-foreground">
                        <div className="h-1.5 w-1.5 rounded-full bg-purple-500" /> ASYNC
                      </div>
                    </div>
                  </div>

                  <div className="relative space-y-6">
                    {/* Connection Line */}
                    <div className="absolute left-6 top-6 bottom-6 w-[2px] bg-gradient-to-b from-blue-500 via-purple-500 to-emerald-500 opacity-20" />

                    {selectedTrace.spans.map((span, idx) => {
                      const service = services.find(s => s.name === span.service) || services[0]
                      return (
                        <motion.div
                          initial={{ opacity: 0, x: -20 }}
                          animate={{ opacity: 1, x: 0 }}
                          transition={{ delay: idx * 0.1 }}
                          key={span.id}
                          className="relative flex items-center gap-6 group"
                        >
                          <div className={`relative z-10 p-3 rounded-full border transition-all group-hover:scale-110 ${service.bg} ${service.color} border-current/20 shadow-lg`}>
                            <service.icon className="h-6 w-6" />
                          </div>

                          <div className="flex-1 bg-card/50 border border-border/50 p-4 rounded-xl backdrop-blur-sm group-hover:border-blue-500/30 transition-all">
                            <div className="flex justify-between items-center mb-1">
                              <span className="text-[10px] font-black uppercase text-muted-foreground tracking-widest">{span.service}</span>
                              <span className="font-mono text-xs font-bold text-blue-400">{span.duration}ms</span>
                            </div>
                            <div className="text-sm font-bold flex items-center gap-2">
                              {span.operation}
                              {span.status === "success" && <CheckCircle2 className="h-3 w-3 text-emerald-500" />}
                            </div>
                          </div>

                          {idx < selectedTrace.spans.length - 1 && (
                            <div className="absolute left-[23px] top-[50px] h-[34px] w-0 border-l-2 border-dashed border-muted-foreground/20" />
                          )}
                        </motion.div>
                      )
                    })}
                  </div>
                </motion.div>
              )}
            </AnimatePresence>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
