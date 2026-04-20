"use client"

import React, { useState } from "react"
import { motion, AnimatePresence } from "framer-motion"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Bot, Sparkles, Send, User, Brain, ShieldAlert } from "lucide-react"

interface Message {
  id: string
  role: "user" | "assistant"
  content: string
  timestamp: string
}

export function AIInvestigator() {
  const [messages, setMessages] = useState<Message[]>([
    {
      id: "1",
      role: "assistant",
      content: "I am the FinCore AI Investigator. I have analyzed 2.4M transactions. How can I help you today?",
      timestamp: new Date().toLocaleTimeString()
    }
  ])
  const [input, setInput] = useState("")

  const handleSend = () => {
    if (!input.trim()) return

    const userMsg: Message = {
      id: Date.now().toString(),
      role: "user",
      content: input,
      timestamp: new Date().toLocaleTimeString()
    }
    setMessages(prev => [...prev, userMsg])
    setInput("")

    // Simulate AI response
    setTimeout(() => {
      let response = "I've analyzed the request. Based on Heuristic Rule #42 (Velocity), the transaction was flagged due to 5 attempts within 1 second from an unverified IP."
      if (input.toLowerCase().includes("audit")) {
        response = "The Merkle Root Hash is consistent across all 3 geo-regions. No tampering detected in the last 24 hours."
      }
      
      const aiMsg: Message = {
        id: (Date.now() + 1).toString(),
        role: "assistant",
        content: response,
        timestamp: new Date().toLocaleTimeString()
      }
      setMessages(prev => [...prev, aiMsg])
    }, 1000)
  }

  return (
    <Card className="shadow-2xl border-border/50 bg-card/30 backdrop-blur-xl h-[600px] flex flex-col">
      <CardHeader className="bg-purple-500/10 border-b">
        <div className="flex justify-between items-center">
          <div className="flex items-center gap-2">
            <Bot className="h-5 w-5 text-purple-400" />
            <CardTitle className="text-sm font-bold font-mono tracking-widest uppercase">AI_FRAUD_INVESTIGATOR</CardTitle>
          </div>
          <Badge variant="outline" className="text-[10px] bg-purple-500/20 text-purple-300 border-none">COGNITIVE_ENGINE: ACTIVE</Badge>
        </div>
        <CardDescription className="text-[10px] uppercase font-bold tracking-tighter mt-1">LLM-Powered behavioral forensics</CardDescription>
      </CardHeader>
      
      <CardContent className="flex-1 overflow-y-auto p-4 space-y-4">
        <AnimatePresence>
          {messages.map((m) => (
            <motion.div
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              key={m.id}
              className={`flex gap-3 ${m.role === "user" ? "flex-row-reverse" : ""}`}
            >
              <div className={`p-2 rounded-lg max-w-[80%] ${
                m.role === "assistant" 
                ? "bg-muted/50 border border-border/30 text-sm" 
                : "bg-blue-600 text-white text-sm"
              }`}>
                {m.content}
                <div className={`text-[9px] mt-1 opacity-50 ${m.role === "user" ? "text-right" : ""}`}>
                  {m.timestamp}
                </div>
              </div>
            </motion.div>
          ))}
        </AnimatePresence>
      </CardContent>

      <div className="p-4 border-t bg-muted/20">
        <div className="flex gap-2">
          <input
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleSend()}
            placeholder="Ask about transaction risk, audit status..."
            className="flex-1 bg-background/50 border border-border/50 rounded-lg px-4 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-purple-500/50"
          />
          <button 
            onClick={handleSend}
            className="p-2 bg-purple-600 rounded-lg hover:bg-purple-500 transition-colors"
          >
            <Send className="h-4 w-4" />
          </button>
        </div>
      </div>
    </Card>
  )
}
