
import { ReactFlowProvider } from 'reactflow';
import ConversationPage from './npc-conversation-editor';

export default function ConversationFlow() {
  return (
    <ReactFlowProvider>
      <ConversationPage />
    </ReactFlowProvider>
  );
}