import * as SecureStore from 'expo-secure-store';
import { Platform } from 'react-native';

export const apiBaseUrl=(process.env.EXPO_PUBLIC_API_URL||(Platform.OS==='android'?'http://10.0.2.2:8080':'http://localhost:8080')).replace(/\/$/,'');
let refreshPromise:Promise<string|null>|null=null;

async function renewSession():Promise<string|null>{
 if(refreshPromise)return refreshPromise;
 refreshPromise=(async()=>{const refresh=await SecureStore.getItemAsync('refresh_token');if(!refresh)return null;try{const r=await fetch(`${apiBaseUrl}/api/v1/auth/refresh`,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({refresh_token:refresh})});if(!r.ok){await SecureStore.deleteItemAsync('access_token');await SecureStore.deleteItemAsync('refresh_token');return null;}const session=await r.json() as {access_token:string;refresh_token:string};await SecureStore.setItemAsync('access_token',session.access_token);await SecureStore.setItemAsync('refresh_token',session.refresh_token);return session.access_token;}catch{return null;}finally{refreshPromise=null;}})();return refreshPromise;
}

export async function apiFetch(path:string,initialToken:string,init:RequestInit={}):Promise<Response>{
 const token=await SecureStore.getItemAsync('access_token')||initialToken;const headers=new Headers(init.headers);headers.set('Authorization',`Bearer ${token}`);let response=await fetch(`${apiBaseUrl}${path}`,{...init,headers});if(response.status!==401)return response;const renewed=await renewSession();if(!renewed)return response;const retryHeaders=new Headers(init.headers);retryHeaders.set('Authorization',`Bearer ${renewed}`);return fetch(`${apiBaseUrl}${path}`,{...init,headers:retryHeaders});
}
