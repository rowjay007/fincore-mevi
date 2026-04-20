"use client"

import React, { useMemo } from "react"
import { ComposableMap, Geographies, Geography, Line, Marker } from "react-simple-maps"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"

const geoUrl = "https://cdn.jsdelivr.net/npm/world-atlas@2/countries-110m.json"

const regions = [
  { name: "US-EAST-1", coordinates: [-74.006, 40.7128] as [number, number], color: "#3b82f6" },
  { name: "EU-WEST-1", coordinates: [-6.2603, 53.3498] as [number, number], color: "#a855f7" },
  { name: "AP-SOUTH-1", coordinates: [72.8777, 19.076] as [number, number], color: "#10b981" },
]

export function GlobalCommandMap() {
  const connections = useMemo(() => {
    return [
      { from: regions[0].coordinates, to: regions[1].coordinates },
      { from: regions[1].coordinates, to: regions[2].coordinates },
      { from: regions[2].coordinates, to: regions[0].coordinates },
    ]
  }, [])

  return (
    <Card className="shadow-2xl border-border/50 bg-card/30 backdrop-blur-xl overflow-hidden">
      <CardHeader className="bg-muted/10 border-b">
        <div className="flex justify-between items-center">
          <div>
            <CardTitle className="text-sm font-bold font-mono tracking-widest uppercase">Global_Liquidity_Grid</CardTitle>
            <CardDescription className="text-[10px] uppercase font-bold tracking-tighter">Multi-region consensus across 3 continents</CardDescription>
          </div>
          <Badge variant="outline" className="text-[10px] bg-emerald-500/10 text-emerald-400 border-none">QUORUM_LOCKED</Badge>
        </div>
      </CardHeader>
      <CardContent className="p-0 bg-background/40 relative">
        <div className="h-[400px] w-full">
          <ComposableMap projectionConfig={{ scale: 140 }}>
            <Geographies geography={geoUrl}>
              {({ geographies }) =>
                geographies.map((geo) => (
                  <Geography
                    key={geo.rseed}
                    geography={geo}
                    fill="rgba(255, 255, 255, 0.03)"
                    stroke="rgba(255, 255, 255, 0.1)"
                    strokeWidth={0.5}
                  />
                ))
              }
            </Geographies>

            {/* Connection Lines */}
            {connections.map((conn, i) => (
              <Line
                key={i}
                from={conn.from}
                to={conn.to}
                stroke="#3b82f6"
                strokeWidth={1}
                strokeDasharray="4 2"
                opacity={0.3}
              />
            ))}

            {/* Region Markers */}
            {regions.map((region) => (
              <Marker key={region.name} coordinates={region.coordinates}>
                <g>
                  <circle r={4} fill={region.color} className="animate-pulse" />
                  <circle r={12} fill={region.color} opacity={0.1} />
                  <text
                    textAnchor="middle"
                    y={-15}
                    style={{ fontSize: "8px", fontWeight: "bold", fill: "#888", fontFamily: "JetBrains Mono" }}
                  >
                    {region.name}
                  </text>
                </g>
              </Marker>
            ))}
          </ComposableMap>
        </div>

        {/* Overlay Stats */}
        <div className="absolute bottom-4 left-4 right-4 flex justify-between gap-4">
          {regions.map(r => (
            <div key={r.name} className="flex-1 bg-background/60 border border-border/30 p-2 rounded-lg backdrop-blur-md">
              <div className="text-[9px] font-black uppercase text-muted-foreground">{r.name}</div>
              <div className="flex justify-between items-center">
                <span className="text-xs font-mono font-bold text-blue-400">12ms</span>
                <span className="text-[8px] font-bold text-emerald-500">SYNCED</span>
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}
