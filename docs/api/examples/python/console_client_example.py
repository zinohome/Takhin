"""
Takhin Console REST API Client - Python Example

This example demonstrates how to interact with the Takhin Console REST API
using the Python requests library.

Install dependencies:
    pip install requests
"""

import requests
from typing import List, Dict, Optional
import json


class TakhinClient:
    """Client for Takhin Console REST API"""

    def __init__(self, base_url: str = "http://localhost:8080/api", api_key: Optional[str] = None):
        """
        Initialize the client
        
        Args:
            base_url: Base URL of the Console API
            api_key: Optional API key for authentication
        """
        self.base_url = base_url.rstrip("/")
        self.session = requests.Session()
        if api_key:
            self.session.headers["Authorization"] = f"Bearer {api_key}"

    def _request(self, method: str, path: str, **kwargs) -> requests.Response:
        """Make an HTTP request"""
        url = f"{self.base_url}{path}"
        resp = self.session.request(method, url, **kwargs)
        resp.raise_for_status()
        return resp

    # Health endpoints

    def health(self) -> Dict:
        """Get health status"""
        resp = self._request("GET", "/health")
        return resp.json()

    def health_ready(self) -> Dict:
        """Get readiness status"""
        resp = self._request("GET", "/health/ready")
        return resp.json()

    # Topic operations

    def list_topics(self) -> List[Dict]:
        """List all topics"""
        resp = self._request("GET", "/topics")
        return resp.json()

    def get_topic(self, name: str) -> Dict:
        """Get topic details"""
        resp = self._request("GET", f"/topics/{name}")
        return resp.json()

    def create_topic(self, name: str, partitions: int) -> Dict:
        """Create a new topic"""
        data = {"name": name, "partitions": partitions}
        resp = self._request("POST", "/topics", json=data)
        return resp.json()

    def delete_topic(self, name: str) -> None:
        """Delete a topic"""
        self._request("DELETE", f"/topics/{name}")

    # Message operations

    def get_messages(
        self,
        topic: str,
        partition: int,
        offset: int,
        limit: int = 100
    ) -> List[Dict]:
        """
        Get messages from a topic partition
        
        Args:
            topic: Topic name
            partition: Partition ID
            offset: Starting offset
            limit: Maximum number of messages to return
        """
        params = {
            "partition": partition,
            "offset": offset,
            "limit": limit
        }
        resp = self._request("GET", f"/topics/{topic}/messages", params=params)
        return resp.json()

    def produce_message(
        self,
        topic: str,
        partition: int,
        key: str,
        value: str
    ) -> Dict:
        """
        Produce a message to a topic
        
        Args:
            topic: Topic name
            partition: Partition ID
            key: Message key
            value: Message value
        """
        data = {
            "partition": partition,
            "key": key,
            "value": value
        }
        resp = self._request("POST", f"/topics/{topic}/messages", json=data)
        return resp.json()

    # Consumer group operations

    def list_consumer_groups(self) -> List[Dict]:
        """List all consumer groups"""
        resp = self._request("GET", "/consumer-groups")
        return resp.json()

    def get_consumer_group(self, group_id: str) -> Dict:
        """Get consumer group details"""
        resp = self._request("GET", f"/consumer-groups/{group_id}")
        return resp.json()


def main():
    """Example usage"""
    
    # Initialize client (set api_key if authentication is enabled)
    client = TakhinClient(api_key=None)

    print("=== Health Check ===")
    health = client.health()
    print(f"Status: {health['status']}\n")

    # Create topic
    print("=== Create Topic ===")
    topic_name = "events"
    try:
        result = client.create_topic(topic_name, 3)
        print(f"Topic '{topic_name}' created: {result}")
    except requests.HTTPError as e:
        print(f"Warning: {e}")
    print()

    # List topics
    print("=== List Topics ===")
    topics = client.list_topics()
    for topic in topics:
        print(f"Topic: {topic['name']}, Partitions: {topic['partitionCount']}")
        for partition in topic.get('partitions', []):
            print(f"  Partition {partition['id']}: HWM={partition['highWaterMark']}")
    print()

    # Produce messages
    print("=== Produce Messages ===")
    messages = [
        {"key": "user-1", "value": json.dumps({"action": "login", "timestamp": "2024-01-01T10:00:00Z"})},
        {"key": "user-2", "value": json.dumps({"action": "purchase", "amount": 99.99})},
        {"key": "user-1", "value": json.dumps({"action": "logout", "timestamp": "2024-01-01T11:00:00Z"})},
    ]

    for msg in messages:
        try:
            result = client.produce_message(topic_name, 0, msg["key"], msg["value"])
            print(f"Produced: key={msg['key']}, offset={result['offset']}, partition={result['partition']}")
        except requests.HTTPError as e:
            print(f"Failed to produce: {e}")
    print()

    # Read messages
    print("=== Read Messages ===")
    read_messages = client.get_messages(topic_name, partition=0, offset=0, limit=10)
    for msg in read_messages:
        print(f"Offset {msg['offset']}: key={msg['key']}, value={msg['value']}")
    print()

    # List consumer groups
    print("=== List Consumer Groups ===")
    groups = client.list_consumer_groups()
    for group in groups:
        print(f"Group: {group['groupId']}, State: {group['state']}, Members: {group['members']}")
    print()

    # Get consumer group details
    if groups:
        print("=== Consumer Group Details ===")
        group_id = groups[0]['groupId']
        details = client.get_consumer_group(group_id)
        print(f"Group: {details['groupId']}")
        print(f"State: {details['state']}")
        print(f"Protocol: {details.get('protocol', 'N/A')}")
        print("Members:")
        for member in details.get('members', []):
            print(f"  {member['memberId']} ({member['clientId']})")
        print("Offset Commits:")
        for commit in details.get('offsetCommits', []):
            print(f"  {commit['topic']}[{commit['partition']}] = {commit['offset']}")
        print()

    print("All examples completed!")


if __name__ == "__main__":
    main()
