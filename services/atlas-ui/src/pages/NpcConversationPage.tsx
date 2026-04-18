
import { ReactFlowProvider } from 'reactflow';
import ConversationPage from './npc-conversation-editor';

export function NpcConversationPage() {
  return (
    <ReactFlowProvider>
      <ConversationPage />
    </ReactFlowProvider>
  );
}