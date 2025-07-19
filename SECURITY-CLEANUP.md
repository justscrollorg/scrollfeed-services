# Security Cleanup Script for YouTube API Key

## 🚨 URGENT: Hardcoded API Key Found
Found hardcoded YouTube API key: AIzaSyAxjy4ug-kVR943vFXL-BSf_J67mkEs-iA

## 🛡️ Required Actions BEFORE Making Repo Public:

### 1. IMMEDIATELY Revoke the Compromised API Key
- Go to Google Cloud Console: https://console.cloud.google.com/
- Navigate to APIs & Services > Credentials
- Find the YouTube API key: AIzaSyAxjy4ug-kVR943vFXL-BSf_J67mkEs-iA
- DELETE or REGENERATE this key immediately
- Create a new API key and update your Kubernetes secret

### 2. Update Kubernetes Secret with New Key
```bash
kubectl delete secret youtube-api-secret
kubectl create secret generic youtube-api-secret --from-literal=YOUTUBE_API_KEY="YOUR_NEW_API_KEY"
```

### 3. Clean Git History (Choose ONE method):

#### Method A: Git Filter-Repo (Recommended)
```bash
# Install git-filter-repo
pip install git-filter-repo

# Create replacement file
echo "AIzaSyAxjy4ug-kVR943vFXL-BSf_J67mkEs-iA" > secrets.txt

# Remove from all history
git filter-repo --replace-text secrets.txt

# Force push clean history
git push --force-with-lease origin main
```

#### Method B: Create New Repository (Safest)
```bash
# Create new repo with clean current state
git checkout main
git branch main-backup  # Backup current branch
git checkout --orphan clean-main
git add .
git commit -m "Initial clean commit without secrets"
git branch -D main
git branch -m clean-main main
git push --force-with-lease origin main
```

### 4. Verify Cleanup
```bash
# Search for any remaining secrets
git log --all --full-history | grep -i "AIza"
git grep -r "AIza" .
```

## 🔒 Security Best Practices Going Forward:

1. **Never commit secrets** - Use environment variables and Kubernetes secrets
2. **Use .env files** (excluded in .gitignore)  
3. **Pre-commit hooks** to scan for secrets
4. **Regular secret rotation**
5. **GitHub secret scanning** (enable when public)

## ⚠️ WARNING:
The compromised API key is visible in commits:
- 4aa37e1: yt
- 72059f1: yt

DO NOT make this repository public until the git history is cleaned!
