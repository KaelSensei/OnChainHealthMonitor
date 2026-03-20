# 🧰 Tools Explained - OnChainHealthMonitor

> **Who this is for:** Someone who wants to understand the project but has never heard of Go, Prometheus, or Kubernetes. No experience required. We'll explain everything.

---

## 📖 Introduction

OnChainHealthMonitor is a system that watches over DeFi (Decentralised Finance) protocols in real time - checking their health, detecting anomalies, and firing alerts when something goes wrong. Think of it as a 24/7 monitoring control room for blockchain services.

To do all of this reliably, the project uses a lot of tools - each one solving a specific problem. This might feel overwhelming at first, but every tool earns its place. This guide explains each one in plain language, with analogies, so you can build a mental model before touching any code.

> 🇫🇷 **Français**
>
> OnChainHealthMonitor est un système qui surveille en temps réel les protocoles DeFi (Finance Décentralisée) - vérifiant leur santé, détectant les anomalies et envoyant des alertes quand quelque chose cloche. Imaginez une salle de contrôle ouverte 24h/24 pour des services blockchain.
>
> Pour faire tout cela de manière fiable, le projet utilise beaucoup d'outils - chacun résolvant un problème précis. Cela peut sembler écrasant au début, mais chaque outil a sa raison d'être. Ce guide explique chacun en langage simple, avec des analogies, pour que vous puissiez comprendre l'ensemble avant de toucher au code.

---

## 1. 🐹 Go (Golang)

**What is it?**

Go (also called Golang) is a programming language created by Google in 2009. It's the language all the services in this project are written in.

**Why Go for microservices?**

Go is designed to be fast, simple, and efficient with memory. It compiles to a tiny binary (a single executable file) that starts up instantly - perfect for containers. It has built-in support for running many things at the same time (called "goroutines"), which is essential when you're monitoring thousands of blockchain events per second.

In one sentence: **Go is like Python's simplicity but with C's speed** - you write readable code that runs incredibly fast.

**Why not Python or Node.js?**

Python is wonderful for scripts and data science, but it's slower and uses more memory. Node.js is great for web APIs but wasn't designed for the heavy concurrent workloads monitoring systems face. Go was literally designed for this use case.

> 🇫🇷 **Français**
>
> **C'est quoi ?**
>
> Go (aussi appelé Golang) est un langage de programmation créé par Google en 2009. C'est le langage dans lequel tous les services de ce projet sont écrits.
>
> **Pourquoi Go pour les microservices ?**
>
> Go est conçu pour être rapide, simple et efficace en mémoire. Il compile en un minuscule binaire (un seul fichier exécutable) qui démarre instantanément - parfait pour les conteneurs. Il a un support natif pour exécuter de nombreuses choses en même temps (appelé "goroutines"), ce qui est essentiel quand on surveille des milliers d'événements blockchain par seconde.
>
> En une phrase : **Go, c'est la simplicité de Python avec la vitesse du C** - vous écrivez du code lisible qui s'exécute incroyablement vite.
>
> **Pourquoi pas Python ou Node.js ?**
>
> Python est merveilleux pour les scripts, mais plus lent. Node.js est excellent pour les APIs web mais n'a pas été conçu pour les lourdes charges concurrentes que les systèmes de monitoring affrontent. Go a littéralement été conçu pour ce cas d'usage.

---

## 2. 🐳 Docker

**What is a container?**

Imagine you're sending a cake to a friend. Instead of just sending the recipe, you send the cake already baked, in a sealed box, with everything it needs. That's a container - your application, its dependencies, its runtime, all packaged together.

**The "works on my machine" problem**

You've written code that works perfectly on your laptop. But when you deploy it to a server, it crashes. Different OS version, different library versions, different configuration. Docker solves this by bundling everything together. If it runs in the container on your laptop, it runs the same way everywhere.

**What is docker-compose?**

OnChainHealthMonitor isn't one app - it's many services (an API gateway, a metrics collector, a blockchain watcher, etc.). `docker-compose` is like a conductor: it reads a single `docker-compose.yml` file and starts all those containers in the right order, connecting them together on a shared network.

> 🇫🇷 **Français**
>
> **C'est quoi un conteneur ?**
>
> Imaginez que vous envoyez un gâteau à un ami. Au lieu d'envoyer juste la recette, vous envoyez le gâteau déjà cuit, dans une boîte scellée, avec tout ce qu'il faut. C'est un conteneur - votre application, ses dépendances, son environnement, tout empaqueté ensemble.
>
> **Le problème du "ça marche sur ma machine"**
>
> Votre code fonctionne parfaitement sur votre ordinateur, mais plante en production. Versions d'OS différentes, bibliothèques différentes. Docker résout cela en tout regroupant. Si ça tourne dans le conteneur chez vous, ça tourne pareil partout.
>
> **C'est quoi docker-compose ?**
>
> OnChainHealthMonitor n'est pas une seule appli - c'est plusieurs services. `docker-compose` est comme un chef d'orchestre : il lit un fichier `docker-compose.yml` et démarre tous ces conteneurs dans le bon ordre, les connectant sur un réseau partagé.

---

## 3. 📊 Prometheus

**What is a metric?**

A metric is a number measured over time. "How many requests per second is this service handling?" - that's a metric. "How much memory is being used?" - metric. "How many blockchain blocks have been processed?" - also a metric.

**How Prometheus works**

Prometheus is like a security camera - but instead of recording video, it records numbers. Every few seconds, Prometheus visits each service's `/metrics` endpoint (a special URL that returns a list of numbers), reads the values, and stores them in its own database with a timestamp. This is called **scraping**.

**What does it store?**

It stores time-series data: the same metric measured again and again over time. "At 14:00:00, there were 342 requests/sec. At 14:00:15, there were 367 requests/sec." This lets you graph trends, detect spikes, and set alerts.

> 🇫🇷 **Français**
>
> **C'est quoi une métrique ?**
>
> Une métrique, c'est un nombre mesuré dans le temps. "Combien de requêtes par seconde ce service traite-t-il ?" - c'est une métrique. "Quelle quantité de mémoire est utilisée ?" - métrique. Prometheus est comme une caméra de surveillance - mais au lieu d'enregistrer de la vidéo, il enregistre des chiffres.
>
> **Comment Prometheus fonctionne**
>
> Toutes les quelques secondes, Prometheus visite l'endpoint `/metrics` de chaque service (une URL spéciale qui retourne une liste de chiffres), lit les valeurs et les stocke dans sa propre base de données avec un horodatage. On appelle ça le **scraping** (grattage).
>
> **Qu'est-ce qu'il stocke ?**
>
> Il stocke des données de séries temporelles : la même métrique mesurée encore et encore dans le temps. "À 14h00:00, il y avait 342 req/sec. À 14h00:15, 367 req/sec." Cela permet de tracer des tendances, détecter des pics et configurer des alertes.

---

## 4. 📈 Grafana

**What is Grafana?**

If Prometheus is the camera recording numbers, Grafana is the monitor where you watch the footage. Grafana is a dashboarding tool that connects to Prometheus and turns raw numbers into beautiful, interactive graphs and charts.

**What do dashboards look like?**

Picture a big screen on a wall covered in colourful graphs: CPU usage over time, number of active DeFi protocols being monitored, API response times, error rates. Each panel is a live-updating visualisation. You can zoom in, change time ranges, and drill down into specific services.

**What is PromQL?**

PromQL (Prometheus Query Language) is the language you use to ask Prometheus questions - in one sentence: **PromQL is like SQL for your metrics**, letting you write queries like "show me the average request latency for the last 5 minutes, grouped by service."

> 🇫🇷 **Français**
>
> **C'est quoi Grafana ?**
>
> Si Prometheus est la caméra qui enregistre des chiffres, Grafana est le moniteur où vous regardez les images. Grafana est un outil de tableaux de bord qui se connecte à Prometheus et transforme les chiffres bruts en graphiques beaux et interactifs.
>
> **À quoi ressemblent les dashboards ?**
>
> Imaginez un grand écran couvert de graphiques colorés : utilisation CPU dans le temps, nombre de protocoles DeFi surveillés, temps de réponse des APIs, taux d'erreurs. Chaque panneau est une visualisation qui se met à jour en direct.
>
> **C'est quoi PromQL ?**
>
> PromQL (Prometheus Query Language) est le langage pour interroger Prometheus - en une phrase : **PromQL est comme le SQL pour vos métriques**, vous permettant d'écrire des requêtes comme "montrez-moi la latence moyenne des 5 dernières minutes, groupée par service."

---

## 5. 🔭 OpenTelemetry (OTel)

**Three pillars: traces, metrics, logs**

Observability (knowing what's happening inside your system) rests on three pillars:

- 📏 **Metrics** - numbers over time (we covered those above)
- 📝 **Logs** - text messages from your app ("User X made a request at 14:02")
- 🧵 **Traces** - a complete record of one request's journey through multiple services

**Why does distributed tracing matter?**

In a microservices system, one user action might touch 5 different services. If something is slow, which service is the culprit? Tracing answers this: it follows a single request as it hops from service to service, measuring how long each hop takes. Without tracing, finding a performance bottleneck is like finding a needle in a haystack.

**What is OpenTelemetry?**

OpenTelemetry is an open standard - a shared language - for generating and sending traces, metrics, and logs from your code, regardless of which monitoring backend you use.

**How it's implemented in this project**

All four Go services use the OpenTelemetry Go SDK (`go.opentelemetry.io/otel`). Each service initialises a tracer at startup that exports spans over **OTLP gRPC** to the OTel Collector at `otel-collector:4317`. If the collector is unreachable, the services degrade gracefully - they log a warning and continue running without tracing.

Spans currently instrumented:

| Service | Span name | What it measures |
|---------|-----------|-----------------|
| `collector` | `generate_event` | Time to generate and emit one mock DeFi event |
| `analyzer` | `analyze_protocol` | Time to compute a health score for one protocol |
| `notifier` | _(planned)_ | Alert evaluation loop |
| `api` | _(planned)_ | HTTP handler duration per endpoint |

Each span carries attributes like `protocol.id`, `event.type`, and `health.score` so you can filter by DeFi protocol directly in the Jaeger UI. See `docs/development/TRACING_GUIDE.md` for how to add spans to new code.

> 🇫🇷 **Français**
>
> **Trois piliers : traces, métriques, logs**
>
> L'observabilité repose sur trois piliers :
>
> - 📏 **Métriques** - des chiffres dans le temps
> - 📝 **Logs** - des messages texte de votre appli
> - 🧵 **Traces** - l'enregistrement complet du parcours d'une requête à travers plusieurs services
>
> **Pourquoi le tracing distribué est-il important ?**
>
> Dans un système de microservices, une action utilisateur peut toucher 5 services différents. Si quelque chose est lent, quel service est coupable ? Le tracing répond à cette question : il suit une requête de service en service, mesurant le temps de chaque étape. Sans tracing, trouver un goulot d'étranglement, c'est comme chercher une aiguille dans une botte de foin.
>
> **C'est quoi OpenTelemetry ?**
>
> OpenTelemetry est un standard ouvert - un langage commun - pour générer et envoyer des traces, métriques et logs depuis votre code, quel que soit le backend de monitoring utilisé.
>
> **Comment c'est implémenté dans ce projet**
>
> Les quatre services Go utilisent le SDK OpenTelemetry Go. Chaque service initialise un traceur au démarrage qui exporte les spans en **OTLP gRPC** vers l'OTel Collector à `otel-collector:4317`. Si le collecteur est inaccessible, les services dégradent gracieusement - ils journalisent un avertissement et continuent sans traçage. Les spans instrumentés incluent `generate_event` (collector) et `analyze_protocol` (analyzer), avec des attributs comme `protocol.id` et `health.score` pour filtrer par protocole DeFi dans l'interface Jaeger.

---

## 6. 🔍 Jaeger

**What is Jaeger?**

If OpenTelemetry is the language services use to report their traces, Jaeger is the backend that receives, stores, and lets you explore those traces. Think of Jaeger as the post office that receives and organises all the "journey reports" sent by your services.

**The trace pipeline in this project**

Traces don't travel directly from your Go services to Jaeger. There's an **OTel Collector** in the middle - it receives spans from the services, batches them, and forwards them to Jaeger:

```
Go service  →  otel-collector:4317 (gRPC)  →  jaeger:4317  →  Jaeger UI
```

This decouples your services from the storage backend. Swapping Jaeger for Grafana Tempo, for example, would only require changing one line in `observability/otel/otel-collector-config.yaml`.

**How to use the Jaeger UI**

Open **http://localhost:16686** in your browser. To find traces:

1. Select a **Service** from the dropdown (e.g. `onchain-collector`, `onchain-analyzer`)
2. Optionally filter by **Operation** (e.g. `generate_event`, `analyze_protocol`)
3. Click **Find Traces** - each result row is one complete trace
4. Click a trace to open the span waterfall timeline

If the data fetcher took 800ms and everything else took 5ms, the waterfall makes this instantly obvious.

**The zpages debug endpoint**

The OTel Collector exposes a debug UI at **http://localhost:55679** (zpages). This shows live stats on spans received, batches exported, and any pipeline errors - useful if traces aren't appearing in Jaeger.

> 🇫🇷 **Français**
>
> **C'est quoi Jaeger ?**
>
> Si OpenTelemetry est le langage que les services utilisent pour rapporter leurs traces, Jaeger est le backend qui reçoit, stocke et vous permet d'explorer ces traces. Pensez à Jaeger comme la poste qui reçoit et organise tous les "rapports de voyage" envoyés par vos services.
>
> **Le pipeline de traces dans ce projet**
>
> Les traces ne voyagent pas directement des services Go vers Jaeger. Un **OTel Collector** fait l'intermédiaire - il reçoit les spans des services, les regroupe par lots et les transmet à Jaeger. Ce découplage signifie que remplacer Jaeger par Grafana Tempo ne nécessite qu'une modification d'une ligne dans `observability/otel/otel-collector-config.yaml`.
>
> **Comment utiliser l'interface Jaeger**
>
> Ouvrez **http://localhost:16686** dans votre navigateur. Sélectionnez un **Service** dans le menu déroulant (par ex. `onchain-collector`), filtrez optionnellement par opération (`generate_event`, `analyze_protocol`), cliquez sur **Find Traces**, puis cliquez sur une trace pour ouvrir la vue cascade des spans.
>
> **L'endpoint de débogage zpages**
>
> L'OTel Collector expose une interface de débogage à **http://localhost:55679** (zpages). Elle affiche les statistiques en direct sur les spans reçus, les lots exportés et toute erreur dans le pipeline - utile si les traces n'apparaissent pas dans Jaeger.

---

## 7. ⚙️ GitHub Actions

**What is CI/CD?**

CI/CD stands for Continuous Integration / Continuous Deployment. It's a pipeline that automatically runs tasks every time code is pushed to the repository. Think of it as a robot assistant that jumps into action the moment you push new code.

**Lint → Test → Build → Push**

- 🔍 **Lint** - checks your code for style and obvious errors (`go vet` + `staticcheck`)
- ✅ **Test** - runs all automated tests with the race detector (`go test ./... -race`)
- 🏗️ **Build** - compiles the code and creates Docker images
- 📦 **Push** - ships the Docker image to GHCR (GitHub Container Registry)

**Seven workflows in this project**

This project has seven workflow files in `.github/workflows/`:

| File | What it does |
|------|-------------|
| `ci-api.yml` | Lint, test, build, push the `api` service |
| `ci-collector.yml` | Lint, test, build, push the `collector` service |
| `ci-analyzer.yml` | Lint, test, build, push the `analyzer` service |
| `ci-notifier.yml` | Lint, test, build, push the `notifier` service |
| `ci-infra.yml` | Validate docker-compose, Kong config, OpenAPI spec |
| `release.yml` | Build all 4 services on a git tag push (`v*.*.*`) |
| `pr-checks.yml` | Commitlint + markdownlint on every pull request |

**Path-based triggers - the smart part**

In a monorepo with four services, you don't want to rebuild everything when only one service changes. Each service workflow watches only its own directory:

```yaml
on:
  push:
    paths:
      - 'services/collector/**'
```

Change a file in `services/collector/` → only the collector workflow runs. The other three services are completely untouched. This saves time and prevents false image versions.

**GHCR - where the images go**

Images are pushed to **GHCR** (GitHub Container Registry) at:

```
ghcr.io/kaelsensei/onchainhealthmonitor/<service>:latest
ghcr.io/kaelsensei/onchainhealthmonitor/<service>:sha-<commit>
```

GHCR is free for public repos and uses the built-in `GITHUB_TOKEN` - no separate secret to configure. Pull any image with:

```bash
docker pull ghcr.io/kaelsensei/onchainhealthmonitor/api:latest
```

**For more detail, see [../deployment/CI_CD_GUIDE.md](../deployment/CI_CD_GUIDE.md).**

> 🇫🇷 **Français**
>
> **C'est quoi le CI/CD ?**
>
> CI/CD signifie Intégration Continue / Déploiement Continu. C'est un pipeline qui exécute automatiquement des tâches à chaque fois que du code est poussé dans le dépôt. Imaginez un assistant robot qui entre en action dès que vous poussez du nouveau code.
>
> **Lint → Test → Build → Push**
>
> - 🔍 **Lint** - vérifie le style et les erreurs évidentes (`go vet` + `staticcheck`)
> - ✅ **Test** - exécute tous les tests avec le détecteur de races (`go test ./... -race`)
> - 🏗️ **Build** - compile le code et crée des images Docker
> - 📦 **Push** - envoie l'image Docker vers GHCR (GitHub Container Registry)
>
> **Sept workflows dans ce projet**
>
> Ce projet dispose de sept fichiers workflow dans `.github/workflows/` :
>
> - `ci-api.yml`, `ci-collector.yml`, `ci-analyzer.yml`, `ci-notifier.yml` - lint + test + build + push par service
> - `ci-infra.yml` - valide docker-compose, la config Kong, et la spec OpenAPI
> - `release.yml` - build de tous les services quand un tag `v*.*.*` est poussé
> - `pr-checks.yml` - commitlint + markdownlint sur chaque pull request
>
> **Déclencheurs par chemin - la partie intelligente**
>
> Dans un monorepo avec quatre services, on ne veut pas tout reconstruire quand un seul service change. Chaque workflow de service surveille uniquement son propre répertoire. Modifier un fichier dans `services/collector/` → seul le workflow collector s'exécute. Les trois autres services restent intacts.
>
> **GHCR - là où vont les images**
>
> Les images sont poussées vers **GHCR** (GitHub Container Registry) à l'adresse `ghcr.io/kaelsensei/onchainhealthmonitor/<service>:latest`. GHCR est gratuit pour les dépôts publics et utilise le `GITHUB_TOKEN` intégré - aucun secret séparé à configurer.
>
> **Pour plus de détails, voir [../deployment/CI_CD_GUIDE.md](../deployment/CI_CD_GUIDE.md).**

---

## 8. 🦍 Kong (API Gateway)

**The problem: exposing microservices directly**

Imagine you have 10 different microservices. Without a gateway, each would need its own public URL, its own authentication system, its own rate limiting. That's 10 times the work and 10 times the attack surface.

**What is an API Gateway?**

Kong is the single front door of the entire system. All traffic from the outside world hits Kong first. Kong then decides where to route each request. It's like a hotel receptionist - guests don't wander the building freely; they check in at the desk and are directed to the right room.

**Rate limiting and auth plugins**

Kong has a plugin system. The **rate limiting plugin** prevents abuse by capping how many requests a single client can make per minute. The **auth plugin** requires a valid API key or JWT token before any request is forwarded. These protections would otherwise need to be built into every single service.

> 🇫🇷 **Français**
>
> **Le problème : exposer les microservices directement**
>
> Imaginez 10 microservices différents. Sans gateway, chacun aurait besoin de sa propre URL publique, son propre système d'authentification, sa propre limitation de débit. C'est 10 fois le travail et 10 fois la surface d'attaque.
>
> **C'est quoi une API Gateway ?**
>
> Kong est la porte d'entrée unique de tout le système. Tout le trafic extérieur passe par Kong en premier, qui décide où router chaque requête. C'est comme le réceptionniste d'un hôtel - les clients ne se promènent pas librement dans le bâtiment ; ils s'enregistrent à l'accueil et sont dirigés vers la bonne chambre.
>
> **Plugins de rate limiting et d'auth**
>
> Kong dispose d'un système de plugins. Le plugin de **rate limiting** empêche les abus en limitant le nombre de requêtes par minute. Le plugin **auth** exige une clé API ou un token JWT valide avant de transférer toute requête. Ces protections devraient sinon être intégrées dans chaque service.

---

## 9. 🏗️ Terraform

**What is Infrastructure as Code?**

Instead of clicking around in the AWS console to create servers, databases, and networks, you write code that describes what you want. Terraform reads that code and creates the infrastructure for you. This is called **Infrastructure as Code (IaC)**.

**Why is clicking in the AWS console bad?**

Because it's invisible, unrepeatable, and error-prone. If you click to create a server today and need to recreate it in 6 months, you won't remember every setting. With Terraform, the entire infrastructure is in version-controlled files - reviewable, shareable, and reproducible.

**What does a `.tf` file declare?**

A `.tf` file (Terraform file) describes resources: "I want a GKE cluster in europe-west1, with this VPC, with workload identity and autoscaling." Terraform figures out the order to create things and handles dependencies automatically.

**How it's implemented in this project**

`infra/terraform/` provisions the full GKE stack on GCP using two reusable modules:

- **`modules/networking/`** - creates the VPC and subnet (`10.0.0.0/16`) in `europe-west1`, with secondary ranges for Pod and Service IPs
- **`modules/gke/`** - provisions the GKE cluster with workload identity, shielded nodes, and a node pool that autoscales between 1 and 5 `e2-medium` nodes

Copy `infra/terraform/terraform.tfvars.example` to `terraform.tfvars`, fill in your GCP project ID, and run `terraform init && terraform apply`. The output provides the exact `gcloud` command to configure `kubectl`. See [../deployment/INFRASTRUCTURE_GUIDE.md](../deployment/INFRASTRUCTURE_GUIDE.md) for the full walkthrough.

> 🇫🇷 **Français**
>
> **C'est quoi l'Infrastructure as Code ?**
>
> Au lieu de cliquer dans la console GCP pour créer des serveurs et des bases de données, vous écrivez du code qui décrit ce que vous voulez. Terraform lit ce code et crée l'infrastructure pour vous. C'est ce qu'on appelle l'**Infrastructure as Code (IaC)**.
>
> **Pourquoi cliquer dans la console est-il mauvais ?**
>
> Parce que c'est invisible, non reproductible et sujet aux erreurs. Avec Terraform, toute l'infrastructure est dans des fichiers versionnés - révisables, partageables et reproductibles.
>
> **Comment c'est implémenté dans ce projet**
>
> `infra/terraform/` utilise deux modules réutilisables - **networking** (VPC + sous-réseau en europe-west1) et **gke** (cluster GKE avec workload identity, nœuds blindés, autoscaling 1–5 nœuds). Copiez `terraform.tfvars.example` en `terraform.tfvars`, renseignez votre project ID GCP, puis lancez `terraform init && terraform apply`. La sortie fournit la commande `gcloud` exacte pour configurer `kubectl`.

---

## 10. ☸️ Kubernetes (K8s)

**What is container orchestration?**

Docker lets you run one container. But in production, you might need 50 containers across 10 servers, with automatic restarts when containers crash, automatic scaling when traffic spikes, and rolling updates so deploys don't cause downtime. Kubernetes does all of this.

**Plain English definitions**

- 🫛 **Pod** - the smallest unit in K8s; usually one container (like a single running copy of your app)
- 📦 **Deployment** - tells K8s "I want 3 copies of this Pod running at all times; if one crashes, start a new one"
- 🌐 **Service** - gives a stable address to a group of Pods so other services can reach them even as Pods restart
- 📈 **HPA** - Horizontal Pod Autoscaler; automatically adds or removes Pod replicas based on CPU or memory load

**Why isn't docker-compose enough for production?**

docker-compose runs on one machine and has no health checks, auto-healing, or multi-server support. Kubernetes was designed for exactly the challenges of production: scale, reliability, and zero-downtime deploys.

**How it's implemented in this project**

All workloads live in the `onchain-health-monitor` namespace. `infra/k8s/` contains:

- `namespace.yaml` - declares the namespace
- `prometheus-config.yaml` - ConfigMap that configures the Prometheus scrape settings inside the cluster
- Four `ServiceMonitor` resources (one per service) - these are custom resources consumed by the Prometheus Operator that automatically discover and scrape each service's `/metrics` endpoint as pods restart and get new IPs

The `api` service has an **HPA** defined in its Helm chart that scales between 2 and 10 replicas when CPU exceeds 70%. All four services are reachable via ClusterIP Services on their respective ports (8080–8083).

> 🇫🇷 **Français**
>
> **C'est quoi l'orchestration de conteneurs ?**
>
> Docker vous permet d'exécuter un conteneur. Mais en production, vous pourriez avoir besoin de 50 conteneurs sur 10 serveurs, avec des redémarrages automatiques quand un conteneur plante, une mise à l'échelle automatique lors des pics de trafic, et des mises à jour progressives. Kubernetes fait tout cela.
>
> **Définitions en langage simple**
>
> - 🫛 **Pod** - la plus petite unité dans K8s ; généralement un conteneur (comme une seule copie en cours d'exécution de votre appli)
> - 📦 **Deployment** - dit à K8s "Je veux 3 copies de ce Pod actives en permanence ; si l'une plante, en démarrer une nouvelle"
> - 🌐 **Service** - donne une adresse stable à un groupe de Pods pour que d'autres services puissent les atteindre même quand les Pods redémarrent
> - 📈 **HPA** - Horizontal Pod Autoscaler ; ajoute ou supprime automatiquement des répliques de Pod en fonction de la charge CPU ou mémoire
>
> **Pourquoi docker-compose ne suffit-il pas en production ?**
>
> docker-compose tourne sur une machine et n'a pas de vérifications de santé, d'auto-guérison ou de support multi-serveurs. Kubernetes a été conçu exactement pour les défis de la production : échelle, fiabilité et déploiements sans interruption.
>
> **Comment c'est implémenté dans ce projet**
>
> Toutes les charges de travail vivent dans le namespace `onchain-health-monitor`. `infra/k8s/` contient le namespace, une ConfigMap Prometheus, et quatre ressources `ServiceMonitor` (une par service) pour la découverte automatique des endpoints `/metrics` par l'opérateur Prometheus. Le service `api` dispose d'un HPA qui s'adapte entre 2 et 10 répliques quand le CPU dépasse 70%.

---

## 11. ⛵ Helm

**The problem with raw Kubernetes YAML**

Kubernetes is configured with YAML files. A real application might have 20+ YAML files, full of repeated values (the app version, the Docker image tag, the resource limits). When you want to deploy the same app to staging and production with slightly different settings, you end up with duplicate files that drift out of sync.

**What is Helm?**

Helm is the package manager for Kubernetes - like `apt` for Ubuntu or `npm` for Node.js. It lets you package all your Kubernetes YAML files into a **Chart** (a reusable bundle), with variables that can be customised per environment.

**What is a Chart?**

A Chart is a folder containing templated Kubernetes files plus a `values.yaml` file where you set the variables. Deploying to production? Set `replicas: 5`. Deploying to staging? Set `replicas: 1`. Same Chart, different values. No copy-pasting.

**How it's implemented in this project**

`infra/helm/` uses the **umbrella chart pattern**: one parent chart (`onchain-health-monitor/`) that declares the four per-service charts as dependencies. This means a single `helm install` deploys all four services atomically.

Each subchart (api, collector, analyzer, notifier) contains a Deployment, Service, HPA, and ConfigMap template. Images are pulled from `ghcr.io/kaelsensei/onchainhealthmonitor/<service>:latest` - the same registry that GitHub Actions pushes to on every merge.

To deploy:

```bash
cd infra/helm
helm dep update onchain-health-monitor   # resolves subchart dependencies
helm install onchain-health-monitor ./onchain-health-monitor \
  --namespace onchain-health-monitor --create-namespace
```

To roll out a new image: `helm upgrade onchain-health-monitor ./onchain-health-monitor --set api.image.tag=sha-abc1234`

> 🇫🇷 **Français**
>
> **Le problème avec le YAML Kubernetes brut**
>
> Kubernetes est configuré avec des fichiers YAML. Une vraie application peut avoir 20+ fichiers YAML, remplis de valeurs répétées. Quand vous voulez déployer la même appli en staging et en production avec des paramètres légèrement différents, vous finissez avec des fichiers dupliqués qui divergent.
>
> **C'est quoi Helm ?**
>
> Helm est le gestionnaire de paquets pour Kubernetes - comme `apt` pour Ubuntu ou `npm` pour Node.js. Il vous permet de regrouper tous vos fichiers YAML Kubernetes en un **Chart** (un paquet réutilisable), avec des variables personnalisables par environnement.
>
> **C'est quoi un Chart ?**
>
> Un Chart est un dossier contenant des fichiers Kubernetes templatisés et un fichier `values.yaml` où vous définissez les variables. Déployer en production ? Définissez `replicas: 5`. En staging ? `replicas: 1`. Même Chart, valeurs différentes. Pas de copier-coller.
>
> **Comment c'est implémenté dans ce projet**
>
> `infra/helm/` utilise le **pattern umbrella chart** : un chart parent (`onchain-health-monitor/`) qui déclare les quatre charts par service comme dépendances. Un seul `helm install` déploie les quatre services de façon atomique. Chaque sous-chart contient un Deployment, un Service, un HPA et un ConfigMap. Les images viennent de `ghcr.io/kaelsensei/onchainhealthmonitor/<service>:latest`. Pour résoudre les dépendances avant le déploiement : `helm dep update onchain-health-monitor`.

---

## 12. 🚨 Grafana Alerting

**What is an alerting rule?**

An alerting rule is a condition written in PromQL that Grafana checks continuously. Example: "If the number of failed blockchain RPC calls exceeds 100 per minute for more than 2 minutes, fire an alert." When the condition becomes true, Grafana notifies you via Slack, email, PagerDuty, or whatever channel you've configured.

**What is an SLO?**

An SLO (Service Level Objective) is a target you set for your system's reliability. For example: "This API must respond successfully 99.9% of the time." Grafana Alerting helps you track whether you're meeting your SLOs and alerts you the moment you start breaking them.

**What happens when an alert fires?**

1. Grafana detects the rule condition is true
2. It sends a notification to the configured contact point (Slack, email, etc.)
3. The on-call engineer investigates using the Grafana dashboard and Jaeger traces
4. Once fixed, the alert resolves automatically and sends an "all clear" message

> 🇫🇷 **Français**
>
> **C'est quoi une règle d'alerte ?**
>
> Une règle d'alerte est une condition écrite en PromQL que Grafana vérifie en continu. Exemple : "Si le nombre d'appels RPC blockchain échoués dépasse 100 par minute pendant plus de 2 minutes, déclencher une alerte." Quand la condition devient vraie, Grafana vous notifie via Slack, email ou tout autre canal configuré.
>
> **C'est quoi un SLO ?**
>
> Un SLO (Service Level Objective) est un objectif de fiabilité que vous fixez pour votre système. Par exemple : "Cette API doit répondre avec succès 99,9% du temps." Grafana Alerting vous aide à suivre si vous respectez vos SLOs et vous alerte dès que vous commencez à les enfreindre.
>
> **Que se passe-t-il quand une alerte se déclenche ?**
>
> 1. Grafana détecte que la condition de la règle est vraie
> 2. Il envoie une notification au point de contact configuré (Slack, email, etc.)
> 3. L'ingénieur de garde enquête avec le dashboard Grafana et les traces Jaeger
> 4. Une fois corrigé, l'alerte se résout automatiquement et envoie un message "tout est clair"

---

## 🔗 How It All Fits Together

Here's the full data flow in plain language:

A user (or an automated script) sends a request to the system. That request first hits **Kong**, the API gateway, which checks authentication and applies rate limiting before forwarding it to the correct microservice. The microservice - written in **Go** - does its work: maybe it fetches health data from a DeFi protocol on the blockchain. As it processes the request, the Go service records an **OpenTelemetry** trace (the journey log) and increments its **Prometheus** metrics (the counters). The trace is shipped via the **OTel Collector** to **Jaeger**, where an engineer can later visualise exactly how long each step took. The metrics are scraped by Prometheus every few seconds, and **Grafana** turns those metrics into live dashboards. Meanwhile, **Grafana Alerting** is watching those dashboards: if any metric crosses a danger threshold, it fires an alert to the on-call team.

All of this runs inside **Docker** containers, orchestrated by **Kubernetes** so that crashed containers are automatically restarted and traffic is load-balanced. **Helm** packages the Kubernetes configuration neatly, making it easy to deploy to different environments. The cloud infrastructure itself (the servers, networks, and storage that host all of this) is defined in **Terraform** files, so the entire environment can be recreated from scratch with a single command. And every time a developer pushes new code, **GitHub Actions** automatically lints it, runs the tests, builds new Docker images, and deploys the updated services - keeping the whole system continuously up to date without manual intervention.

> 🇫🇷 **Français**
>
> Voici le flux de données complet en langage simple :
>
> Un utilisateur (ou un script automatisé) envoie une requête au système. Cette requête touche d'abord **Kong**, la gateway API, qui vérifie l'authentification et applique la limitation de débit avant de la transmettre au bon microservice. Le microservice - écrit en **Go** - fait son travail : peut-être qu'il récupère des données de santé d'un protocole DeFi sur la blockchain. En traitant la requête, le service Go enregistre une trace **OpenTelemetry** (le journal de parcours) et incrémente ses métriques **Prometheus** (les compteurs). La trace est envoyée via l'**OTel Collector** à **Jaeger**, où un ingénieur peut visualiser exactement combien de temps chaque étape a pris. Les métriques sont collectées par Prometheus toutes les quelques secondes, et **Grafana** transforme ces métriques en tableaux de bord en direct. Pendant ce temps, **Grafana Alerting** surveille ces tableaux de bord : si une métrique dépasse un seuil de danger, il envoie une alerte à l'équipe de permanence.
>
> Tout cela tourne dans des conteneurs **Docker**, orchestrés par **Kubernetes** pour que les conteneurs en panne soient automatiquement redémarrés et que le trafic soit réparti. **Helm** regroupe soigneusement la configuration Kubernetes, facilitant le déploiement dans différents environnements. L'infrastructure cloud elle-même (les serveurs, réseaux et stockage) est définie dans des fichiers **Terraform**, permettant de recréer tout l'environnement depuis zéro avec une seule commande. Et à chaque fois qu'un développeur pousse du nouveau code, **GitHub Actions** le lint automatiquement, lance les tests, construit de nouvelles images Docker et déploie les services mis à jour - gardant tout le système continuellement à jour sans intervention manuelle.

---

*Document generated for the OnChainHealthMonitor project. For setup instructions, see [GETTING_STARTED.md](GETTING_STARTED.md). For architecture decisions, see [../architecture/DECISIONS.md](../architecture/DECISIONS.md).*
