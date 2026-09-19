from flask import Flask
import requests
from opentelemetry import trace
from opentelemetry.sdk.resources import SERVICE_NAME, Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.instrumentation.flask import FlaskInstrumentor
from opentelemetry.instrumentation.requests import RequestsInstrumentor

trace.set_tracer_provider(
    TracerProvider(
        resource=Resource.create({SERVICE_NAME: "service-a"})
    )
)

otlp_exporter = OTLPSpanExporter(
    endpoint="http://simplest-collector.default.svc.cluster.local:4317",
    insecure=True,
)
trace.get_tracer_provider().add_span_processor(
    BatchSpanProcessor(otlp_exporter)
)

app = Flask(__name__)
FlaskInstrumentor().instrument_app(app)
RequestsInstrumentor().instrument()

@app.route("/random-num")
def random_num():
    with trace.get_tracer(__name__).start_as_current_span("fetch-random-num"):
        res = requests.get("http://service-b:8080/random-num", timeout=5)
        return res.text
