'use client';

import * as React from 'react';
import { AlertTriangle, RefreshCw, Bug } from 'lucide-react';
import { errorLogger } from '@/services/errorLogger';

interface GlobalErrorProps {
  error: Error & { digest?: string };
  reset: () => void;
}

/**
 * Global error page component for handling critical application errors
 * This page is shown when an error occurs that can't be handled by regular error boundaries
 * Note: This component must include its own styling as it may render when the app is broken
 */
export default function GlobalError({ error, reset }: GlobalErrorProps) {
  const [detailsExpanded, setDetailsExpanded] = React.useState(false);
  const [reportSent, setReportSent] = React.useState(false);
  const [reportLoading, setReportLoading] = React.useState(false);

  React.useEffect(() => {
    // Log error automatically when the component mounts
    if (typeof window !== 'undefined') {
      console.error('Global Error:', error);
      
      // Log to error reporting service
      errorLogger.logError(error).catch((logError) => {
        console.warn('Failed to log global error:', logError);
      });
    }
  }, [error]);

  const handleReportError = async () => {
    if (reportSent || reportLoading) return;
    
    setReportLoading(true);
    try {
      await errorLogger.logError(error);
      setReportSent(true);
    } catch (reportError) {
      console.error('Failed to send error report:', reportError);
    } finally {
      setReportLoading(false);
    }
  };

  const handleReload = () => {
    if (typeof window !== 'undefined') {
      window.location.href = '/';
    }
  };

  return (
    <html>
      <body>
        <div
          style={{
            display: 'flex',
            minHeight: '100vh',
            alignItems: 'center',
            justifyContent: 'center',
            padding: '1rem',
            fontFamily: 'system-ui, -apple-system, sans-serif',
            backgroundColor: '#fafafa',
            color: '#171717',
          }}
        >
          <div
            style={{
              width: '100%',
              maxWidth: '32rem',
              backgroundColor: 'white',
              borderRadius: '0.75rem',
              border: '1px solid #e5e5e5',
              boxShadow: '0 1px 3px 0 rgba(0, 0, 0, 0.1)',
            }}
          >
            <div
              style={{
                textAlign: 'center',
                padding: '1.5rem 1.5rem 0',
              }}
            >
              <div
                style={{
                  margin: '0 auto 1rem',
                  display: 'flex',
                  height: '3rem',
                  width: '3rem',
                  alignItems: 'center',
                  justifyContent: 'center',
                  borderRadius: '50%',
                  backgroundColor: '#fef2f2',
                }}
              >
                <AlertTriangle
                  style={{
                    height: '1.5rem',
                    width: '1.5rem',
                    color: '#dc2626',
                  }}
                />
              </div>
              <h1
                style={{
                  fontSize: '1.25rem',
                  fontWeight: '600',
                  margin: '0 0 1rem',
                }}
              >
                Critical Application Error
              </h1>
            </div>
            
            <div style={{ padding: '0 1.5rem 1.5rem' }}>
              <div
                style={{
                  padding: '1rem',
                  backgroundColor: '#fef2f2',
                  border: '1px solid #fecaca',
                  borderRadius: '0.5rem',
                  marginBottom: '1rem',
                }}
              >
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    marginBottom: '0.5rem',
                  }}
                >
                  <AlertTriangle
                    style={{
                      height: '1rem',
                      width: '1rem',
                      color: '#dc2626',
                      marginRight: '0.5rem',
                    }}
                  />
                  <h2
                    style={{
                      fontSize: '0.875rem',
                      fontWeight: '600',
                      margin: 0,
                    }}
                  >
                    Something went wrong
                  </h2>
                </div>
                <p
                  style={{
                    fontSize: '0.875rem',
                    margin: 0,
                    color: '#7f1d1d',
                  }}
                >
                  {error.message || 'A critical error occurred that prevented the application from loading properly.'}
                </p>
              </div>

              {process.env.NODE_ENV === 'development' && (
                <div style={{ marginBottom: '1rem' }}>
                  <button
                    onClick={() => setDetailsExpanded(!detailsExpanded)}
                    style={{
                      width: '100%',
                      padding: '0.5rem 1rem',
                      fontSize: '0.875rem',
                      backgroundColor: 'white',
                      border: '1px solid #d1d5db',
                      borderRadius: '0.375rem',
                      cursor: 'pointer',
                    }}
                  >
                    {detailsExpanded ? 'Hide' : 'Show'} Technical Details
                  </button>
                  
                  {detailsExpanded && (
                    <div
                      style={{
                        marginTop: '0.5rem',
                        padding: '0.75rem',
                        backgroundColor: '#f5f5f5',
                        borderRadius: '0.375rem',
                        fontSize: '0.75rem',
                        fontFamily: 'monospace',
                        wordBreak: 'break-all',
                      }}
                    >
                      <div style={{ fontWeight: '500', marginBottom: '0.5rem' }}>
                        Error Details:
                      </div>
                      <div style={{ marginBottom: '0.5rem' }}>
                        <span style={{ color: '#6b7280' }}>Name:</span> {error.name}
                      </div>
                      <div style={{ marginBottom: '0.5rem' }}>
                        <span style={{ color: '#6b7280' }}>Message:</span> {error.message}
                      </div>
                      {error.digest && (
                        <div style={{ marginBottom: '0.5rem' }}>
                          <span style={{ color: '#6b7280' }}>Digest:</span> {error.digest}
                        </div>
                      )}
                      {error.stack && (
                        <div style={{ marginBottom: '0.5rem' }}>
                          <span style={{ color: '#6b7280' }}>Stack:</span>
                          <pre
                            style={{
                              marginTop: '0.25rem',
                              whiteSpace: 'pre-wrap',
                              fontSize: '0.75rem',
                            }}
                          >
                            {error.stack}
                          </pre>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              )}

              <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
                <button
                  onClick={handleReportError}
                  disabled={reportSent || reportLoading}
                  style={{
                    width: '100%',
                    padding: '0.5rem 1rem',
                    fontSize: '0.875rem',
                    backgroundColor: reportSent || reportLoading ? '#f3f4f6' : 'white',
                    border: '1px solid #d1d5db',
                    borderRadius: '0.375rem',
                    cursor: reportSent || reportLoading ? 'not-allowed' : 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                  }}
                >
                  <Bug
                    style={{
                      height: '1rem',
                      width: '1rem',
                      marginRight: '0.5rem',
                      ...(reportLoading ? { animation: 'spin 1s linear infinite' } : {}),
                    }}
                  />
                  {reportLoading ? 'Sending Report...' : reportSent ? 'Report Sent' : 'Report This Error'}
                </button>
                
                <div style={{ display: 'flex', gap: '0.5rem' }}>
                  <button
                    onClick={reset}
                    style={{
                      flex: 1,
                      padding: '0.5rem 1rem',
                      fontSize: '0.875rem',
                      backgroundColor: '#171717',
                      color: 'white',
                      border: 'none',
                      borderRadius: '0.375rem',
                      cursor: 'pointer',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                    }}
                  >
                    <RefreshCw
                      style={{
                        height: '1rem',
                        width: '1rem',
                        marginRight: '0.5rem',
                      }}
                    />
                    Try Again
                  </button>
                  <button
                    onClick={handleReload}
                    style={{
                      flex: 1,
                      padding: '0.5rem 1rem',
                      fontSize: '0.875rem',
                      backgroundColor: 'white',
                      border: '1px solid #d1d5db',
                      borderRadius: '0.375rem',
                      cursor: 'pointer',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                    }}
                  >
                    Reload App
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </body>
    </html>
  );
}