import { useCallback, useMemo } from 'react'
import { useParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  ReactFlow, Controls, Background, BackgroundVariant,
  useNodesState, useEdgesState, addEdge,
  type Node, type Edge, type Connection,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { apiClient } from '@/lib/api'
import type { Agent, PendingHire } from '@/lib/types'
import { useAppStore } from '@/store/useAppStore'
import { AgentNode } from './AgentNode'
import { GhostNode } from './GhostNode'
import { DetailPanel } from './DetailPanel'

const NODE_TYPES = { agentNode: AgentNode, ghostNode: GhostNode } as const

function buildFlow(
  agents: Agent[],
  hires: PendingHire[],
  onApprove: (id: string) => void,
  onReject: (id: string) => void,
): { nodes: Node[]; edges: Edge[] } {
  const nodes: Node[] = [
    ...agents.map((a, i) => ({
      id: a.id, type: 'agentNode' as const,
      position: { x: (i % 4) * 180, y: Math.floor(i / 4) * 110 },
      data: { agent: a },
    })),
    ...hires.map((h, i) => ({
      id: `ghost-${h.id}`, type: 'ghostNode' as const,
      position: { x: agents.length * 180 + (i % 3) * 180, y: Math.floor(i / 3) * 130 },
      data: {
        hire: h,
        onApprove: () => onApprove(h.id),
        onReject: () => onReject(h.id),
      },
    })),
  ]

  const edges: Edge[] = [
    ...agents.filter((a) => a.manager_id).map((a) => ({
      id: `${a.manager_id}-${a.id}`, source: a.manager_id!,
      target: a.id, style: { stroke: '#3f3f46' },
    })),
    ...hires.map((h) => ({
      id: `ghost-edge-${h.id}`,
      source: h.reporting_to_agent_id,
      target: `ghost-${h.id}`,
      style: { stroke: '#3f3f46', strokeDasharray: '5 4' },
    })),
  ]

  return { nodes, edges }
}

export function OrgChart() {
  const { companyId } = useParams<{ companyId: string }>()
  const qc = useQueryClient()
  const selectedAgentId = useAppStore((s) => s.agentId)
  const setAgentId = useAppStore((s) => s.setAgentId)

  const { data: agents = [] } = useQuery<Agent[]>({
    queryKey: ['agents', companyId],
    queryFn: () => apiClient.get(`/api/companies/${companyId}/agents`).then((r) => r.data),
    enabled: !!companyId,
  })

  const { data: rawHires = [] } = useQuery<PendingHire[]>({
    queryKey: ['hires', companyId],
    queryFn: () => apiClient.get(`/api/companies/${companyId}/hires`).then((r) => r.data),
    enabled: !!companyId,
  })
  const pendingHires = (Array.isArray(rawHires) ? rawHires : []).filter((h) => h.status === 'pending')

  const approve = useMutation({
    mutationFn: (hireId: string) => apiClient.post(`/api/companies/${companyId}/hires/${hireId}/approve`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['hires', companyId] })
      qc.invalidateQueries({ queryKey: ['agents', companyId] })
    },
  })
  const reject = useMutation({
    mutationFn: (hireId: string) => apiClient.post(`/api/companies/${companyId}/hires/${hireId}/reject`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['hires', companyId] }),
  })

  const { nodes: initNodes, edges: initEdges } = useMemo(
    () => buildFlow(agents, pendingHires, (id) => approve.mutate(id), (id) => reject.mutate(id)),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [agents, pendingHires],
  )
  const [nodes, , onNodesChange] = useNodesState(initNodes)
  const [edges, setEdges, onEdgesChange] = useEdgesState(initEdges)

  const reassign = useMutation({
    mutationFn: ({ agentId, newManagerId }: { agentId: string; newManagerId: string }) =>
      apiClient.post(`/api/companies/${companyId}/agents/${agentId}/reassign`, { new_manager_id: newManagerId }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['agents', companyId] }),
  })
  void reassign

  const onConnect = useCallback((connection: Connection) => setEdges((eds) => addEdge(connection, eds)), [setEdges])
  const onNodeDragStop = useCallback((_: unknown, node: Node) => { void node }, [])
  const selectedAgent = agents.find((a) => a.id === selectedAgentId) ?? null

  return (
    <div className="flex h-full">
      <div className="flex-1 h-full">
        <ReactFlow nodes={nodes} edges={edges} nodeTypes={NODE_TYPES}
          onNodesChange={onNodesChange} onEdgesChange={onEdgesChange}
          onConnect={onConnect} onNodeDragStop={onNodeDragStop}
          onNodeClick={(_, node) => {
            if (!node.id.startsWith('ghost-')) setAgentId(node.id)
          }}
          onPaneClick={() => setAgentId(null)} fitView className="bg-zinc-950">
          <Controls className="[&>button]:bg-zinc-800 [&>button]:border-zinc-700 [&>button]:text-zinc-300" />
          <Background variant={BackgroundVariant.Dots} gap={24} size={1} color="#27272a" />
        </ReactFlow>
      </div>
      {selectedAgent && <DetailPanel agent={selectedAgent} onClose={() => setAgentId(null)} />}
    </div>
  )
}
