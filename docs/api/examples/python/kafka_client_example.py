"""
Takhin Kafka Protocol Client - Python Example

This example demonstrates how to interact with Takhin using the kafka-python library.

Install dependencies:
    pip install kafka-python
"""

from kafka import KafkaProducer, KafkaConsumer, KafkaAdminClient
from kafka.admin import NewTopic, ConfigResource, ConfigResourceType
from kafka.errors import TopicAlreadyExistsError
import json
import time


BOOTSTRAP_SERVERS = ['localhost:9092']
TOPIC_NAME = 'demo-topic'


def create_topic():
    """Create a topic using Kafka Admin API"""
    admin_client = KafkaAdminClient(bootstrap_servers=BOOTSTRAP_SERVERS)
    
    topic = NewTopic(
        name=TOPIC_NAME,
        num_partitions=3,
        replication_factor=1
    )
    
    try:
        admin_client.create_topics([topic])
        print(f"Topic '{TOPIC_NAME}' created successfully")
    except TopicAlreadyExistsError:
        print(f"Topic '{TOPIC_NAME}' already exists")
    finally:
        admin_client.close()


def produce_messages():
    """Produce messages to a topic"""
    producer = KafkaProducer(
        bootstrap_servers=BOOTSTRAP_SERVERS,
        key_serializer=lambda k: k.encode('utf-8') if k else None,
        value_serializer=lambda v: json.dumps(v).encode('utf-8'),
        acks='all',  # Wait for all ISR replicas
        retries=3
    )
    
    messages = [
        {"key": "user-1", "value": {"action": "login", "timestamp": "2024-01-01T10:00:00Z"}},
        {"key": "user-2", "value": {"action": "purchase", "amount": 99.99}},
        {"key": "user-1", "value": {"action": "logout", "timestamp": "2024-01-01T11:00:00Z"}},
    ]
    
    print("\n=== Producing Messages ===")
    for msg in messages:
        future = producer.send(
            TOPIC_NAME,
            key=msg["key"],
            value=msg["value"],
            partition=0
        )
        
        # Wait for send to complete
        record_metadata = future.get(timeout=10)
        print(f"Produced: key={msg['key']}, offset={record_metadata.offset}, "
              f"partition={record_metadata.partition}")
    
    producer.flush()
    producer.close()


def consume_messages():
    """Consume messages from a topic"""
    consumer = KafkaConsumer(
        TOPIC_NAME,
        bootstrap_servers=BOOTSTRAP_SERVERS,
        auto_offset_reset='earliest',
        enable_auto_commit=False,
        consumer_timeout_ms=5000,  # Stop after 5 seconds of no messages
        value_deserializer=lambda v: json.loads(v.decode('utf-8'))
    )
    
    print("\n=== Consuming Messages ===")
    for message in consumer:
        print(f"Consumed: partition={message.partition}, offset={message.offset}, "
              f"key={message.key.decode('utf-8') if message.key else None}, "
              f"value={message.value}")
    
    consumer.close()


def consumer_group_example():
    """Example using consumer groups"""
    GROUP_ID = 'demo-consumer-group'
    
    consumer = KafkaConsumer(
        TOPIC_NAME,
        bootstrap_servers=BOOTSTRAP_SERVERS,
        group_id=GROUP_ID,
        auto_offset_reset='earliest',
        enable_auto_commit=True,
        auto_commit_interval_ms=1000,
        value_deserializer=lambda v: json.loads(v.decode('utf-8'))
    )
    
    print(f"\n=== Consumer Group '{GROUP_ID}' ===")
    count = 0
    for message in consumer:
        print(f"Group message: partition={message.partition}, offset={message.offset}, "
              f"key={message.key.decode('utf-8') if message.key else None}")
        count += 1
        if count >= 3:
            break
    
    consumer.close()


def describe_topic():
    """Describe topic configuration"""
    admin_client = KafkaAdminClient(bootstrap_servers=BOOTSTRAP_SERVERS)
    
    # Describe topic configs
    resource = ConfigResource(ConfigResourceType.TOPIC, TOPIC_NAME)
    configs = admin_client.describe_configs([resource])
    
    print(f"\n=== Topic Configuration: {TOPIC_NAME} ===")
    for config_resource, config_entries in configs.items():
        for config_key, config_entry in config_entries.items():
            print(f"  {config_key} = {config_entry.value}")
    
    admin_client.close()


def list_consumer_groups():
    """List all consumer groups"""
    admin_client = KafkaAdminClient(bootstrap_servers=BOOTSTRAP_SERVERS)
    
    groups = admin_client.list_consumer_groups()
    
    print("\n=== Consumer Groups ===")
    for group_id, group_type in groups:
        print(f"  Group: {group_id}, Type: {group_type}")
    
    admin_client.close()


def get_offsets():
    """Get partition offsets"""
    consumer = KafkaConsumer(
        bootstrap_servers=BOOTSTRAP_SERVERS,
        enable_auto_commit=False
    )
    
    # Get partitions for topic
    partitions = consumer.partitions_for_topic(TOPIC_NAME)
    
    print(f"\n=== Partition Offsets for {TOPIC_NAME} ===")
    for partition in sorted(partitions):
        # Get beginning offset
        tp = {'topic': TOPIC_NAME, 'partition': partition}
        consumer.assign([tp])
        consumer.seek_to_beginning(tp)
        beginning = consumer.position(tp)
        
        # Get end offset
        consumer.seek_to_end(tp)
        end = consumer.position(tp)
        
        print(f"  Partition {partition}: beginning={beginning}, end={end}, messages={end-beginning}")
    
    consumer.close()


def transactional_produce():
    """Example of transactional producer"""
    producer = KafkaProducer(
        bootstrap_servers=BOOTSTRAP_SERVERS,
        transactional_id='my-transactional-producer',
        key_serializer=lambda k: k.encode('utf-8') if k else None,
        value_serializer=lambda v: json.dumps(v).encode('utf-8'),
        acks='all',
        enable_idempotence=True
    )
    
    producer.init_transactions()
    
    print("\n=== Transactional Produce ===")
    try:
        producer.begin_transaction()
        
        messages = [
            {"key": "txn-1", "value": {"data": "message 1"}},
            {"key": "txn-2", "value": {"data": "message 2"}},
        ]
        
        for msg in messages:
            producer.send(TOPIC_NAME, key=msg["key"], value=msg["value"])
            print(f"Sent: key={msg['key']}")
        
        producer.commit_transaction()
        print("Transaction committed successfully")
    except Exception as e:
        print(f"Transaction failed: {e}")
        producer.abort_transaction()
    finally:
        producer.close()


def main():
    """Run all examples"""
    print("Takhin Kafka Client Examples\n")
    
    # Create topic
    create_topic()
    time.sleep(1)
    
    # Produce messages
    produce_messages()
    time.sleep(1)
    
    # Consume messages
    consume_messages()
    time.sleep(1)
    
    # Consumer group example
    consumer_group_example()
    time.sleep(1)
    
    # Describe topic
    describe_topic()
    
    # List consumer groups
    list_consumer_groups()
    
    # Get offsets
    get_offsets()
    
    # Transactional produce
    transactional_produce()
    
    print("\nAll examples completed!")


if __name__ == "__main__":
    main()
