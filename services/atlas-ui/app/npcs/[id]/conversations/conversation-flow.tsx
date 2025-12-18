"use client"

import { ReactFlowProvider } from 'reactflow';
import ConversationPage from './page';

export default function ConversationFlow() {
  return (
    <ReactFlowProvider>
      <ConversationPage />
    </ReactFlowProvider>
  );
}