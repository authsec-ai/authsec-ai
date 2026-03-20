// Quick test script to verify the ClientsService is working
import { ClientsService } from './services/clients';

export async function testClientService() {
  try {
    console.log('Testing ClientsService...');
    
    const result = await ClientsService.getClients(
      { workspace_id: 'e47ac10b-58cc-4372-a567-0e02b2c3d480' },
      { page: 1, pageSize: 5 }
    );
    
    console.log('✅ Service test successful!');
    console.log('Total clients:', result.count);
    console.log('Clients with auth methods:');
    
    result.data.forEach(client => {
      console.log(`- ${client.name}: ${client.attachedMethods.length} auth methods`);
      if (client.attachedMethods.length > 0) {
        client.attachedMethods.forEach(method => {
          console.log(`  • ${method.name} ${method.isDefault ? '(default)' : ''}`);
        });
      }
    });
    
    return result;
  } catch (error) {
    console.error('❌ Service test failed:', error);
    throw error;
  }
}

// You can call this in the browser console: testClientService()
(window as any).testClientService = testClientService;