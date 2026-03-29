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
import type { Agent } from '@/lib/types'
import { useAppStore } from '@/store/useAppStore'
import { AgentNode } from './AgentNode'
import { DetailPanel } from './DetailPanel'

const NODE_TYPES = { agentNode: AgentNode } as const

function agentsToFlow(agents: Agent[]): { nodes: Node[]; edges: Edge[] } {
  const nodes: Node[] = agents.map((a, i) => ({
    id: a.id, type: 'agentNode',
    position: { x: (i % 4) * 160, y: Math.floor(i / 4) * 100 },
    data: { agent: a },
  }))
  const edges: Edge[] = agents.filter((a) => a.manager_id).map((a) => ({
    id: `${a.manager_id}-${a.id}`, source: a.manager_id!,
    target: a.id, style: { stroke: '#3f3f46' },
  }))
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

  const { nodes: initNodes, edges: initEdges } = useMemo(() => agentsToFlow(agents), [agents])
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
          onNodeClick={(_, node) => setAgentId(node.id)}
          onPaneClick={() => setAgentId(null)} fitView className="bg-zinc-950">
          <Controls className="[&>button]:bg-zinc-800 [&>button]:border-zinc-700 [&>button]:text-zinc-300" />
          <Background variant={BackgroundVariant.Dots} gap={24} size={1} color="#27272a" />
        </ReactFlow>
      </div>
      {selectedAgent && <DetailPanel agent={selectedAgent} onClose={() => setAgentId(null)} />}
    </div>
  )
}
