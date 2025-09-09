import React, { useState } from 'react';
import { useKeycloak } from '@react-keycloak/web';

interface ReportItem {
  id: string;
  title: string;
  description: string;
}

const ReportPage: React.FC = () => {
  const { keycloak, initialized } = useKeycloak();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [report, setReport] = useState<ReportItem[]>([])

  const downloadReport = async () => {
    if (!keycloak?.token) {
      setError('Not authenticated');
      return;
    }

    try {
      setLoading(true);
      setError(null);

      const response = await fetch(`${process.env.REACT_APP_API_URL}/reports`, {
        headers: {
          'Authorization': `Bearer ${keycloak.token}`
        }
      });

      if (response.status == 200) {
        try {
          //@ts-ignore
          const parsedBody: ReportItem[] = await response.json();
          setReport(parsedBody)

        } catch (err) {
          console.log(err)
        }
      }

      
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred');
    } finally {
      setLoading(false);
    }
  };

  if (!initialized) {
    return <div>Loading...</div>;
  }

  if (!keycloak.authenticated) {
    return (
      <div className="flex flex-col items-center justify-center min-h-screen bg-gray-100">
        <button
          onClick={() => keycloak.login()}
          className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"
        >
          Login
        </button>
      </div>
    );
  }

  return (
    <div className="flex flex-col items-center justify-center min-h-screen bg-gray-100">
      <div className="p-8 bg-white rounded-lg shadow-md">
        <h1 className="text-2xl font-bold mb-6">Usage Reports</h1>
        
        <button
          onClick={downloadReport}
          disabled={loading}
          className={`px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 ${
            loading ? 'opacity-50 cursor-not-allowed' : ''
          }`}
        >
          {loading ? 'Generating Report...' : 'Download Report'}
        </button>

        {report.length > 0 && (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {report.map((item) => (
              <div
                key={item.id}
                className="p-4 border rounded-lg shadow-sm hover:shadow-md transition"
              >
                <h2 className="font-semibold text-lg mb-2">{item.title}</h2>
                <p className="text-gray-700">{item.description}</p>
                <p className="text-sm text-gray-400 mt-2">ID: {item.id}</p>
              </div>
            ))}
          </div>
        )}

        {error && (
          <div className="mt-4 p-4 bg-red-100 text-red-700 rounded">
            {error}
          </div>
        )}
      </div>
    </div>
  );
};

export default ReportPage;