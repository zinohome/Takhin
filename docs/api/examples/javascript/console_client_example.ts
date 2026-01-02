/**
 * Takhin Console REST API Client - JavaScript/TypeScript Example
 * 
 * This example demonstrates how to interact with the Takhin Console REST API
 * using the Fetch API (available in Node.js 18+ and browsers).
 * 
 * For Node.js < 18, install node-fetch:
 *   npm install node-fetch
 */

const BASE_URL = 'http://localhost:8080/api';
const API_KEY = null; // Set if authentication is enabled

interface CreateTopicRequest {
  name: string;
  partitions: number;
}

interface Topic {
  name: string;
  partitionCount: number;
  partitions: Partition[];
}

interface Partition {
  id: number;
  highWaterMark: number;
}

interface ProduceMessageRequest {
  partition: number;
  key: string;
  value: string;
}

interface ProduceMessageResponse {
  offset: number;
  partition: number;
}

interface Message {
  partition: number;
  offset: number;
  key: string;
  value: string;
  timestamp: number;
}

interface ConsumerGroup {
  groupId: string;
  state: string;
  members: number;
}

interface ConsumerGroupDetails {
  groupId: string;
  state: string;
  protocolType: string;
  protocol: string;
  members: GroupMember[];
  offsetCommits: OffsetCommitEntry[];
}

interface GroupMember {
  memberId: string;
  clientId: string;
  clientHost: string;
  partitions: PartitionAssignment[];
}

interface PartitionAssignment {
  topic: string;
  partition: number;
}

interface OffsetCommitEntry {
  topic: string;
  partition: number;
  offset: number;
  metadata: string;
}

class TakhinClient {
  private baseURL: string;
  private apiKey: string | null;

  constructor(baseURL: string = BASE_URL, apiKey: string | null = API_KEY) {
    this.baseURL = baseURL;
    this.apiKey = apiKey;
  }

  private async request<T>(
    method: string,
    path: string,
    body?: any
  ): Promise<T> {
    const headers: Record<string, string> = {};
    
    if (body) {
      headers['Content-Type'] = 'application/json';
    }
    
    if (this.apiKey) {
      headers['Authorization'] = `Bearer ${this.apiKey}`;
    }

    const response = await fetch(`${this.baseURL}${path}`, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });

    if (!response.ok) {
      const error = await response.text();
      throw new Error(`HTTP ${response.status}: ${error}`);
    }

    if (response.status === 204) {
      return null as T;
    }

    return response.json();
  }

  // Health endpoints
  
  async health(): Promise<{ status: string }> {
    return this.request('GET', '/health');
  }

  async healthReady(): Promise<any> {
    return this.request('GET', '/health/ready');
  }

  // Topic operations

  async listTopics(): Promise<Topic[]> {
    return this.request('GET', '/topics');
  }

  async getTopic(name: string): Promise<Topic> {
    return this.request('GET', `/topics/${name}`);
  }

  async createTopic(name: string, partitions: number): Promise<void> {
    await this.request('POST', '/topics', { name, partitions });
  }

  async deleteTopic(name: string): Promise<void> {
    await this.request('DELETE', `/topics/${name}`);
  }

  // Message operations

  async getMessages(
    topic: string,
    partition: number,
    offset: number,
    limit: number = 100
  ): Promise<Message[]> {
    const params = new URLSearchParams({
      partition: partition.toString(),
      offset: offset.toString(),
      limit: limit.toString(),
    });
    return this.request('GET', `/topics/${topic}/messages?${params}`);
  }

  async produceMessage(
    topic: string,
    partition: number,
    key: string,
    value: string
  ): Promise<ProduceMessageResponse> {
    return this.request('POST', `/topics/${topic}/messages`, {
      partition,
      key,
      value,
    });
  }

  // Consumer group operations

  async listConsumerGroups(): Promise<ConsumerGroup[]> {
    return this.request('GET', '/consumer-groups');
  }

  async getConsumerGroup(groupId: string): Promise<ConsumerGroupDetails> {
    return this.request('GET', `/consumer-groups/${groupId}`);
  }
}

// Example usage
async function main() {
  const client = new TakhinClient();

  try {
    // Health check
    console.log('=== Health Check ===');
    const health = await client.health();
    console.log(`Status: ${health.status}\n`);

    // Create topic
    console.log('=== Create Topic ===');
    const topicName = 'events';
    try {
      await client.createTopic(topicName, 3);
      console.log(`Topic '${topicName}' created`);
    } catch (error) {
      console.log(`Warning: ${error}`);
    }
    console.log();

    // List topics
    console.log('=== List Topics ===');
    const topics = await client.listTopics();
    topics.forEach(topic => {
      console.log(`Topic: ${topic.name}, Partitions: ${topic.partitionCount}`);
      topic.partitions.forEach(p => {
        console.log(`  Partition ${p.id}: HWM=${p.highWaterMark}`);
      });
    });
    console.log();

    // Produce messages
    console.log('=== Produce Messages ===');
    const messages = [
      { key: 'user-1', value: JSON.stringify({ action: 'login', timestamp: '2024-01-01T10:00:00Z' }) },
      { key: 'user-2', value: JSON.stringify({ action: 'purchase', amount: 99.99 }) },
      { key: 'user-1', value: JSON.stringify({ action: 'logout', timestamp: '2024-01-01T11:00:00Z' }) },
    ];

    for (const msg of messages) {
      const result = await client.produceMessage(topicName, 0, msg.key, msg.value);
      console.log(`Produced: key=${msg.key}, offset=${result.offset}, partition=${result.partition}`);
    }
    console.log();

    // Read messages
    console.log('=== Read Messages ===');
    const readMessages = await client.getMessages(topicName, 0, 0, 10);
    readMessages.forEach(msg => {
      console.log(`Offset ${msg.offset}: key=${msg.key}, value=${msg.value}`);
    });
    console.log();

    // List consumer groups
    console.log('=== List Consumer Groups ===');
    const groups = await client.listConsumerGroups();
    groups.forEach(group => {
      console.log(`Group: ${group.groupId}, State: ${group.state}, Members: ${group.members}`);
    });
    console.log();

    // Get consumer group details
    if (groups.length > 0) {
      console.log('=== Consumer Group Details ===');
      const groupId = groups[0].groupId;
      const details = await client.getConsumerGroup(groupId);
      console.log(`Group: ${details.groupId}`);
      console.log(`State: ${details.state}`);
      console.log(`Protocol: ${details.protocol}`);
      console.log('Members:');
      details.members.forEach(member => {
        console.log(`  ${member.memberId} (${member.clientId})`);
      });
      console.log('Offset Commits:');
      details.offsetCommits.forEach(commit => {
        console.log(`  ${commit.topic}[${commit.partition}] = ${commit.offset}`);
      });
      console.log();
    }

    console.log('All examples completed!');
  } catch (error) {
    console.error('Error:', error);
    process.exit(1);
  }
}

// Run examples
if (require.main === module) {
  main();
}

export { TakhinClient };
