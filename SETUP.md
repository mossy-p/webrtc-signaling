# WebRTC Signaling Server - Setup Instructions

## ✅ What's Been Done

1. **Service Code** - Complete Go implementation at `/Users/moss_p/Documents/GitHub/webrtc-signaling`
2. **GitHub Repository** - Created at https://github.com/mossy-p/webrtc-signaling
3. **Code Pushed** - Main branch pushed (without workflow file)
4. **GitOps Configuration** - Deployed to `moss-server-gitops/applications/webrtc-signaling/`
5. **Redis Password** - Generated: `KBmxn3VcL0/G5eqwcm3xe5fRCtNPfRuDt7hDmoznBvw=`
6. **JWT Secret** - Copied from mossp.me-api (already configured)

## 🔧 Manual Steps Required

### 1. Add GitHub Actions Workflow

The workflow file couldn't be pushed automatically due to PAT scope limitations.

**Option A: Add via GitHub Web Interface**
1. Go to https://github.com/mossy-p/webrtc-signaling
2. Navigate to `.github/workflows/`
3. Click "Add file" → "Create new file"
4. Name it `release.yml`
5. Copy content from `/Users/moss_p/Documents/GitHub/webrtc-signaling/.github/workflows/release.yml`
6. Commit directly to main branch

**Option B: Push with workflow scope**
```bash
cd /Users/moss_p/Documents/GitHub/webrtc-signaling
git push origin main
```
(You'll need to grant `workflow` scope to your GitHub PAT first)

### 2. Seal Redis Password

```bash
# Login to AWS first
aws sso login --profile default

# Seal the generated Redis password
echo -n "KBmxn3VcL0/G5eqwcm3xe5fRCtNPfRuDt7hDmoznBvw=" | kubeseal --raw \
  --from-file=/dev/stdin \
  --namespace mossp-me \
  --name webrtc-signaling-secret

# Copy the output and update:
# moss-server-gitops/applications/webrtc-signaling/sealed-secret.yaml
# Replace REPLACE_WITH_SEALED_REDIS_PASSWORD with the sealed value
```

Then commit and push the GitOps update:
```bash
cd /Users/moss_p/Documents/GitHub/moss-server-gitops
git add applications/webrtc-signaling/sealed-secret.yaml
git commit -m "Add sealed Redis password for webrtc-signaling"
git push
```

### 3. Configure GitHub Secrets

The workflow needs these repository secrets:

1. Go to https://github.com/mossy-p/webrtc-signaling/settings/secrets/actions
2. Add secrets:
   - `GITOPS_REPO` = `mossy-p/moss-server-gitops`
   - `GITOPS_PAT` = Your GitHub PAT with repo and workflow access

### 4. Trigger Deployment

Once the workflow file is added and secrets are configured:

```bash
cd /Users/moss_p/Documents/GitHub/webrtc-signaling
git checkout -b releases/v0.1
git push -u origin releases/v0.1
```

This will:
- Build Docker image
- Push to ghcr.io/mossy-p/webrtc-signaling:0.1.xxx
- Trigger GitOps update
- Deploy to Kubernetes

## 🎯 Testing After Deployment

### 1. Check Pods
```bash
kubectl get pods -n mossp-me | grep webrtc-signaling
kubectl get pods -n mossp-me | grep redis
```

### 2. Check Service
```bash
kubectl get svc -n mossp-me webrtc-signaling
```

### 3. Test Health Endpoint
```bash
curl https://webrtc-signaling.your-tailscale-domain/health
```

### 4. Create a Test Room

From your frontend (with JWT):
```bash
curl -X POST https://webrtc-signaling/api/rooms \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"maxPlayers": 8}'

# Response: {"roomId": "...", "code": "ABCD23"}
```

### 5. Test WebSocket Connection

```javascript
const ws = new WebSocket('wss://webrtc-signaling/ws/signal/ABCD23?displayName=TestPlayer');
ws.onmessage = (e) => console.log('Received:', JSON.parse(e.data));
```

## 📝 Environment Configuration

Once deployed, the service will have:
- **Port**: 8080
- **Environment**: production
- **Allowed Origins**: `https://mossp.me,https://www.mossp.me`
- **Redis**: `redis-service.mossp-me.svc.cluster.local:6379`
- **JWT Secret**: Shared with mossp.me-api
- **Tailscale**: Exposed at `webrtc-signaling.tail-xxxxx.ts.net`

## 🔒 Security Notes

- JWT_SECRET is copied from mossp.me-api (required for token validation)
- Redis password is unique to this service
- Origin filtering prevents unauthorized WebSocket connections
- Room creation requires authentication
- Room joining is public (by design for party games)

## 📊 Monitoring

Check logs:
```bash
kubectl logs -n mossp-me -l app=webrtc-signaling --tail=100 -f
```

Check Redis:
```bash
kubectl exec -it -n mossp-me deployment/redis -- redis-cli -a KBmxn3VcL0/G5eqwcm3xe5fRCtNPfRuDt7hDmoznBvw=
> KEYS room:*
> SMEMBERS room:{room-id}:peers
```

## 🚀 Next Steps

After deployment is verified:
1. Update frontend to use the signaling server
2. Test room creation and joining flows
3. Build the party game frontend
4. Monitor for any issues or performance tuning needed

## 📚 Documentation

- Full API docs: `/Users/moss_p/Documents/GitHub/webrtc-signaling/README.md`
- Example usage in README.md
- WebRTC signaling protocol documented in code comments
