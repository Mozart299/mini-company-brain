"use client";

import { useState, useEffect, useCallback } from "react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Zap, AlertTriangle, Brain, Loader2 } from "lucide-react";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

interface Fact {
  key: string;
  value: unknown;
  source: string;
  version: number;
  created_at: string;
}

interface AlertValue {
  id: string;
  description: string;
  severity: "low" | "medium" | "high";
  detected_at: string;
  evidence: string[];
}

function usePolling<T>(url: string, interval = 5000) {
  const [data, setData] = useState<T | null>(null);
  const load = useCallback(async () => {
    try {
      const res = await fetch(url);
      if (res.ok) setData(await res.json());
    } catch {}
  }, [url]);

  useEffect(() => {
    load();
    const id = setInterval(load, interval);
    return () => clearInterval(id);
  }, [load, interval]);

  return data;
}

const SOURCE_COLORS: Record<string, string> = {
  github: "bg-violet-500/10 text-violet-400 border-violet-500/20",
  slack: "bg-green-500/10 text-green-400 border-green-500/20",
  linear: "bg-blue-500/10 text-blue-400 border-blue-500/20",
  coordinator: "bg-orange-500/10 text-orange-400 border-orange-500/20",
};

const SEVERITY_VARIANTS: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  high: "destructive",
  medium: "default",
  low: "secondary",
};

function IngestionFeed() {
  const facts = usePolling<Fact[]>(`${API}/facts`);
  const recent = (facts ?? [])
    .filter((f) => !f.key.startsWith("drift.alert:"))
    .slice(-30)
    .reverse();

  return (
    <Card className="flex flex-col h-full">
      <CardHeader className="pb-3">
        <CardTitle className="flex items-center gap-2 text-base">
          <Zap className="h-4 w-4 text-yellow-500" />
          Ingestion Feed
        </CardTitle>
        <CardDescription>{recent.length} events stored</CardDescription>
      </CardHeader>
      <CardContent className="flex-1 p-0">
        <ScrollArea className="h-[520px] px-6 pb-6">
          {recent.length === 0 ? (
            <p className="text-sm text-muted-foreground pt-2">No events yet — run <code className="text-xs bg-muted px-1 py-0.5 rounded">make ingest</code></p>
          ) : (
            <div className="flex flex-col gap-2">
              {recent.map((f) => (
                <div key={f.key + f.version} className="rounded-md border bg-muted/30 p-3 text-xs">
                  <div className="flex items-center justify-between gap-2 mb-1.5">
                    <span className="font-medium text-foreground truncate">{f.key}</span>
                    <Badge
                      variant="outline"
                      className={`shrink-0 text-[10px] ${SOURCE_COLORS[f.source] ?? ""}`}
                    >
                      {f.source}
                    </Badge>
                  </div>
                  <pre className="text-muted-foreground whitespace-pre-wrap break-all leading-relaxed">
                    {JSON.stringify(f.value, null, 2)}
                  </pre>
                </div>
              ))}
            </div>
          )}
        </ScrollArea>
      </CardContent>
    </Card>
  );
}

function AlertsPanel() {
  const facts = usePolling<Fact[]>(`${API}/alerts`);
  const alerts = facts ?? [];

  return (
    <Card className="flex flex-col h-full">
      <CardHeader className="pb-3">
        <CardTitle className="flex items-center gap-2 text-base">
          <AlertTriangle className="h-4 w-4 text-orange-500" />
          Drift Alerts
        </CardTitle>
        <CardDescription>
          {alerts.length === 0 ? "No drift detected" : `${alerts.length} active alert${alerts.length > 1 ? "s" : ""}`}
        </CardDescription>
      </CardHeader>
      <CardContent className="flex-1 p-0">
        <ScrollArea className="h-[520px] px-6 pb-6">
          {alerts.length === 0 ? (
            <p className="text-sm text-muted-foreground pt-2">
              Run <code className="text-xs bg-muted px-1 py-0.5 rounded">make coordinator</code> to start detection
            </p>
          ) : (
            <div className="flex flex-col gap-3">
              {alerts.map((a) => {
                const v = a.value as AlertValue;
                return (
                  <div key={a.key} className="rounded-md border bg-muted/30 p-3 space-y-2">
                    <div className="flex items-center justify-between gap-2">
                      <Badge variant={SEVERITY_VARIANTS[v?.severity] ?? "outline"} className="text-[10px]">
                        {v?.severity?.toUpperCase()}
                      </Badge>
                      <span className="text-[10px] text-muted-foreground">
                        {v?.detected_at ? new Date(v.detected_at).toLocaleString() : ""}
                      </span>
                    </div>
                    <p className="text-xs text-foreground leading-relaxed">{v?.description}</p>
                    {v?.evidence?.length > 0 && (
                      <div className="space-y-0.5">
                        {v.evidence.map((e, i) => (
                          <p key={i} className="text-[10px] text-muted-foreground truncate">↳ {e}</p>
                        ))}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </ScrollArea>
      </CardContent>
    </Card>
  );
}

function QueryPanel() {
  const [question, setQuestion] = useState("");
  const [answer, setAnswer] = useState("");
  const [factsUsed, setFactsUsed] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const ask = async () => {
    if (!question.trim() || loading) return;
    setLoading(true);
    setError("");
    setAnswer("");
    try {
      const res = await fetch(`${API}/query`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ question }),
      });
      const data = await res.json();
      if (!res.ok) setError(typeof data === "string" ? data : "Query failed");
      else {
        setAnswer(data.answer);
        setFactsUsed(data.facts_used);
      }
    } catch {
      setError("Could not reach API");
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="flex items-center gap-2 text-base">
          <Brain className="h-4 w-4 text-emerald-500" />
          Ask the Brain
        </CardTitle>
        <CardDescription>Natural language queries over your company knowledge</CardDescription>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="flex gap-2">
          <Input
            placeholder="What decisions were made about auth last sprint?"
            value={question}
            onChange={(e) => setQuestion(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && ask()}
            disabled={loading}
          />
          <Button onClick={ask} disabled={loading || !question.trim()}>
            {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : "Ask"}
          </Button>
        </div>

        {error && <p className="text-xs text-destructive">{error}</p>}

        {answer && (
          <div className="rounded-md border bg-muted/30 p-4 space-y-2">
            <p className="text-sm text-foreground leading-relaxed whitespace-pre-wrap">{answer}</p>
            <p className="text-[10px] text-muted-foreground">Based on {factsUsed} facts</p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

export default function Dashboard() {
  return (
    <div className="max-w-7xl mx-auto px-6 py-8 space-y-6">
      <header>
        <h1 className="text-2xl font-bold tracking-tight">Company Brain</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Distributed knowledge system · ingestion · storage · drift detection · query
        </p>
      </header>
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 items-start">
        <IngestionFeed />
        <AlertsPanel />
        <QueryPanel />
      </div>
    </div>
  );
}
