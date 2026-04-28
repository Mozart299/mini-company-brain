"use client";

import { useState, useEffect, useCallback } from "react";

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

interface AlertFact extends Fact {
  value: AlertValue;
}

const SEVERITY_COLORS: Record<string, string> = {
  high: "text-red-400 border-red-800",
  medium: "text-yellow-400 border-yellow-800",
  low: "text-blue-400 border-blue-800",
};

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

function IngestionFeed() {
  const facts = usePolling<Fact[]>(`${API}/facts`);
  const recent = (facts ?? []).slice(-20).reverse();

  return (
    <section className="flex flex-col gap-2">
      <h2 className="text-sm font-bold text-gray-400 uppercase tracking-widest">
        Ingestion Feed
        <span className="ml-2 text-gray-600 normal-case font-normal">
          ({recent.length} events)
        </span>
      </h2>
      <div className="flex flex-col gap-1 overflow-y-auto max-h-[480px]">
        {recent.length === 0 && (
          <p className="text-gray-600 text-xs">No events yet — run make ingest</p>
        )}
        {recent.map((f) => (
          <div key={f.key + f.version} className="border border-gray-800 rounded p-2 text-xs">
            <div className="flex justify-between items-start gap-2">
              <span className="text-emerald-400 break-all">{f.key}</span>
              <span className="text-gray-500 shrink-0">{f.source}</span>
            </div>
            <pre className="text-gray-500 mt-1 whitespace-pre-wrap break-all">
              {JSON.stringify(f.value, null, 2)}
            </pre>
          </div>
        ))}
      </div>
    </section>
  );
}

function AlertsPanel() {
  const facts = usePolling<AlertFact[]>(`${API}/alerts`);
  const alerts = facts ?? [];

  return (
    <section className="flex flex-col gap-2">
      <h2 className="text-sm font-bold text-gray-400 uppercase tracking-widest">
        Drift Alerts
        <span className="ml-2 text-gray-600 normal-case font-normal">
          ({alerts.length})
        </span>
      </h2>
      <div className="flex flex-col gap-2 overflow-y-auto max-h-[480px]">
        {alerts.length === 0 && (
          <p className="text-gray-600 text-xs">No drift detected — run make coordinator</p>
        )}
        {alerts.map((a) => {
          const v = a.value as AlertValue;
          const color = SEVERITY_COLORS[v?.severity] ?? SEVERITY_COLORS.low;
          return (
            <div key={a.key} className={`border rounded p-3 text-xs ${color}`}>
              <div className="flex justify-between items-center mb-1">
                <span className="uppercase font-bold text-[10px]">{v?.severity}</span>
                <span className="text-gray-500">
                  {v?.detected_at ? new Date(v.detected_at).toLocaleString() : ""}
                </span>
              </div>
              <p className="text-gray-200">{v?.description}</p>
              {v?.evidence?.length > 0 && (
                <div className="mt-2 text-gray-500">
                  {v.evidence.map((e, i) => (
                    <div key={i} className="truncate">↳ {e}</div>
                  ))}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </section>
  );
}

function QueryPanel() {
  const [question, setQuestion] = useState("");
  const [answer, setAnswer] = useState("");
  const [factsUsed, setFactsUsed] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const ask = async () => {
    if (!question.trim()) return;
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
      if (!res.ok) {
        setError(data || "Query failed");
      } else {
        setAnswer(data.answer);
        setFactsUsed(data.facts_used);
      }
    } catch (e) {
      setError("Could not reach API");
    } finally {
      setLoading(false);
    }
  };

  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-sm font-bold text-gray-400 uppercase tracking-widest">
        Ask the Brain
      </h2>
      <div className="flex gap-2">
        <input
          className="flex-1 bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm text-gray-100 placeholder-gray-600 focus:outline-none focus:border-emerald-700"
          placeholder="What decisions were made about auth last sprint?"
          value={question}
          onChange={(e) => setQuestion(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && ask()}
        />
        <button
          onClick={ask}
          disabled={loading}
          className="bg-emerald-700 hover:bg-emerald-600 disabled:bg-gray-800 text-white text-sm px-4 py-2 rounded transition-colors"
        >
          {loading ? "…" : "Ask"}
        </button>
      </div>

      {error && (
        <p className="text-red-400 text-xs">{error}</p>
      )}

      {answer && (
        <div className="border border-gray-800 rounded p-4 text-sm text-gray-200 whitespace-pre-wrap">
          {answer}
          <p className="text-gray-600 text-xs mt-3">Based on {factsUsed} facts</p>
        </div>
      )}
    </section>
  );
}

export default function Dashboard() {
  return (
    <div className="max-w-7xl mx-auto px-6 py-8">
      <header className="mb-8">
        <h1 className="text-2xl font-bold text-emerald-400">Company Brain</h1>
        <p className="text-gray-500 text-sm mt-1">
          Distributed knowledge system — ingestion · storage · drift detection · query
        </p>
      </header>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <IngestionFeed />
        <AlertsPanel />
        <div className="flex flex-col gap-6">
          <QueryPanel />
        </div>
      </div>
    </div>
  );
}
