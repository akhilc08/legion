import { useEffect, useRef, useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Loader2, Send, Pause, Play } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import { apiClient } from '@/lib/api'
import type { ChatMessage } from '@/lib/types'

interface AgentChatProps {
  agentId: string
  companyId: string
}

export function AgentChat({ agentId, companyId }: AgentChatProps) {
  const qc = useQueryClient()
  const [input, setInput] = useState('')
  const [optimistic, setOptimistic] = useState<ChatMessage[]>([])
  const bottomRef = useRef<HTMLDivElement>(null)

  const historyKey = ['chat-history', companyId, agentId]

  const { data: history = [], isLoading: historyLoading } = useQuery<ChatMessage[]>({
    queryKey: historyKey,
    queryFn: async () => {
      const res = await apiClient.get<ChatMessage[]>(
        `/api/companies/${companyId}/agents/${agentId}/chat/history`
      )
      return res.data
    },
  })

  const sendMutation = useMutation({
    mutationFn: async (message: string) => {
      const res = await apiClient.post<{ reply: string }>(
        `/api/companies/${companyId}/agents/${agentId}/chat`,
        { message }
      )
      return res.data.reply
    },
    onMutate: (message) => {
      const userMsg: ChatMessage = {
        role: 'user',
        content: message,
        timestamp: new Date().toISOString(),
      }
      setOptimistic((prev) => [...prev, userMsg])
    },
    onSuccess: (reply) => {
      const assistantMsg: ChatMessage = {
        role: 'assistant',
        content: reply,
        timestamp: new Date().toISOString(),
      }
      setOptimistic((prev) => [...prev, assistantMsg])
      qc.invalidateQueries({ queryKey: historyKey })
    },
    onError: () => {
      // Remove the last optimistic user message on error
      setOptimistic((prev) => prev.slice(0, -1))
    },
  })

  const pauseMutation = useMutation({
    mutationFn: () =>
      apiClient.post(`/api/companies/${companyId}/agents/${agentId}/pause`),
  })

  const resumeMutation = useMutation({
    mutationFn: () =>
      apiClient.post(`/api/companies/${companyId}/agents/${agentId}/resume`),
  })

  const allMessages = [...history, ...optimistic]

  useEffect(() => {
    if (bottomRef.current && typeof bottomRef.current.scrollIntoView === 'function') {
      bottomRef.current.scrollIntoView({ behavior: 'smooth' })
    }
  }, [allMessages.length])

  const handleSend = () => {
    const msg = input.trim()
    if (!msg || sendMutation.isPending) return
    setInput('')
    sendMutation.mutate(msg)
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  return (
    <div className="flex flex-col h-full bg-zinc-950 rounded-lg border border-zinc-800">
      {/* Header */}
      <div className="flex items-center gap-2 px-3 py-2 border-b border-zinc-800">
        <span className="text-xs text-zinc-400 flex-1">Agent Chat</span>
        <Button
          variant="ghost"
          size="sm"
          className="h-7 px-2 text-xs text-zinc-400 hover:text-zinc-100"
          onClick={() => pauseMutation.mutate()}
          disabled={pauseMutation.isPending}
          aria-label="Pause agent"
        >
          <Pause className="h-3 w-3 mr-1" />
          Pause
        </Button>
        <Button
          variant="ghost"
          size="sm"
          className="h-7 px-2 text-xs text-zinc-400 hover:text-zinc-100"
          onClick={() => resumeMutation.mutate()}
          disabled={resumeMutation.isPending}
          aria-label="Resume agent"
        >
          <Play className="h-3 w-3 mr-1" />
          Resume
        </Button>
      </div>

      {/* Messages */}
      <ScrollArea className="flex-1 px-3 py-3">
        {historyLoading ? (
          <div className="flex items-center justify-center h-16">
            <Loader2 className="h-4 w-4 animate-spin text-zinc-500" />
          </div>
        ) : allMessages.length === 0 ? (
          <p className="text-xs text-zinc-600 text-center py-4">No messages yet</p>
        ) : (
          <div className="flex flex-col gap-2">
            {allMessages.map((msg, i) => (
              <div
                key={i}
                className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}
              >
                <div
                  className={`max-w-[80%] rounded-lg px-3 py-2 text-xs leading-relaxed ${
                    msg.role === 'user'
                      ? 'bg-zinc-700 text-zinc-100'
                      : 'bg-zinc-800 text-zinc-200'
                  }`}
                >
                  {msg.content}
                </div>
              </div>
            ))}
            {sendMutation.isPending && (
              <div className="flex justify-start">
                <div className="bg-zinc-800 rounded-lg px-3 py-2">
                  <Loader2 className="h-3 w-3 animate-spin text-zinc-400" />
                </div>
              </div>
            )}
          </div>
        )}
        <div ref={bottomRef} />
      </ScrollArea>

      {/* Input */}
      <div className="flex items-center gap-2 px-3 py-2 border-t border-zinc-800">
        <Input
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Message agent…"
          className="flex-1 h-8 text-xs bg-zinc-900 border-zinc-700 text-zinc-100 placeholder:text-zinc-600 focus-visible:ring-zinc-600"
          disabled={sendMutation.isPending}
        />
        <Button
          size="sm"
          className="h-8 w-8 p-0 bg-zinc-700 hover:bg-zinc-600"
          onClick={handleSend}
          disabled={!input.trim() || sendMutation.isPending}
          aria-label="Send message"
        >
          <Send className="h-3 w-3" />
        </Button>
      </div>
    </div>
  )
}
