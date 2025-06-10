import { NextRequest, NextResponse } from 'next/server';

// Only export dynamic and API functions in development
// This prevents build errors when doing static export for production
const isDevelopment = process.env.NODE_ENV === 'development' || process.env.NEXT_PUBLIC_BUILD_MODE !== 'production';

// Only export dynamic in development to avoid static export issues
export const dynamic = isDevelopment ? 'force-dynamic' : undefined;

// Only export API functions in development to avoid static export issues
export const GET = isDevelopment ? async function(req: NextRequest) {
  console.log('GET /api/bots called');
  
  try {
    // Get auth token and user ID from headers
    const authToken = req.headers.get('Authorization');
    const userID = req.headers.get('X-User-ID') || 'bobrt-user-id';
    
    console.log('Headers:', { authToken, userID });
    
    // Direct backend URL
    const backendUrl = 'https://c9wu2knteb.execute-api.us-east-1.amazonaws.com/prod/bots';
    
    console.log('Calling backend:', backendUrl);
    
    // Forward request to backend
    const response = await fetch(backendUrl, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': authToken || 'Bearer demo-token',
        'X-User-ID': userID,
      },
    });
    
    console.log('Backend response status:', response.status);
    
    if (!response.ok) {
      const errorText = await response.text();
      console.error('Backend error:', errorText);
      return NextResponse.json(
        { message: 'Failed to fetch from backend', error: errorText },
        { status: response.status }
      );
    }
    
    // Get response data
    const responseData = await response.json();
    console.log('Backend response data:', responseData);
    
    // Return the data
    return NextResponse.json(responseData);
  } catch (error) {
    console.error('Bot listing API error:', error);
    return NextResponse.json(
      { message: 'Internal server error', error: error instanceof Error ? error.message : 'Unknown error' },
      { status: 500 }
    );
  }
} : undefined;

export const POST = isDevelopment ? async function(req: NextRequest) {
  console.log('POST /api/bots called');
  
  try {
    const data = await req.json();
    console.log('Request data:', data);
    
    // Get auth token and user ID from headers
    const authToken = req.headers.get('Authorization');
    const userID = req.headers.get('X-User-ID') || 'bobrt-user-id';
    
    console.log('Headers:', { authToken, userID });
    
    // Direct backend URL
    const backendUrl = 'https://c9wu2knteb.execute-api.us-east-1.amazonaws.com/prod/bots';
    
    console.log('Calling backend:', backendUrl);
    
    // Forward request to backend
    const response = await fetch(backendUrl, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': authToken || 'Bearer demo-token',
        'X-User-ID': userID,
      },
      body: JSON.stringify(data),
    });
    
    console.log('Backend response status:', response.status);
    
    if (!response.ok) {
      const errorText = await response.text();
      console.error('Backend error:', errorText);
      
      // For now, let's see the full response and headers to debug
      console.log('Full response:', {
        status: response.status,
        statusText: response.statusText,
        headers: Object.fromEntries(response.headers.entries()),
        body: errorText
      });
      
      return NextResponse.json(
        { 
          message: 'Failed to create bot', 
          error: errorText,
          status: response.status,
          statusText: response.statusText,
          debug: 'Check logs for full response details'
        },
        { status: response.status }
      );
    }
    
    // Get response data
    const responseData = await response.json();
    console.log('Backend response data:', responseData);
    
    // Return the data
    return NextResponse.json(responseData);
  } catch (error) {
    console.error('Bot creation API error:', error);
    return NextResponse.json(
      { message: 'Internal server error', error: error instanceof Error ? error.message : 'Unknown error' },
      { status: 500 }
    );
  }
} : undefined;