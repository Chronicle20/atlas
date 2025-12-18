"use client"

import { ReactFlowProvider } from 'reactflow';
import ConversationPage from './conversation-page';

export default function Page() {
  return (
    <ReactFlowProvider>
      <ConversationPage />
    </ReactFlowProvider>
  );
}