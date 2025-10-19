# How to Upload Wiki to GitHub

GitHub Wiki needs to be initialized manually before it can be cloned. Follow these steps:

## Step 1: Create the First Wiki Page

1. **Go to your repository wiki page:**
   ```
   https://github.com/reaandrew/politest/wiki
   ```

2. **Click "Create the first page"**

3. **Enter the following:**
   - **Title**: `Home`
   - **Content**: Copy and paste the entire content from `docs/wiki/Home.md`

4. **Click "Save Page"**

✅ **The wiki is now initialized!**

## Step 2: Clone Wiki Repository and Upload All Pages

Now that the wiki exists, you can clone it and push all pages at once:

```bash
# Navigate to your project directory
cd /home/andy-rea/development/politest

# Clone the wiki repository (this will work now)
git clone https://github.com/reaandrew/politest.wiki.git wiki-repo

# Copy all wiki markdown files
cp docs/wiki/*.md wiki-repo/

# Commit and push
cd wiki-repo
git add *.md
git commit -m "Add comprehensive wiki documentation"
git push

# Clean up
cd ..
rm -rf wiki-repo
```

**Done!** All 8 wiki pages are now published.

## Alternative: Upload Pages Manually (One by One)

If you prefer to upload manually without git:

### 2. Upload _Sidebar (Navigation)

1. Go to: https://github.com/reaandrew/politest/wiki/_Sidebar/_edit
2. Paste content from `docs/wiki/_Sidebar.md`
3. Click "Save Page"

### 3. Upload Installation-and-Setup

1. Click "New Page"
2. Title: `Installation-and-Setup`
3. Paste content from `docs/wiki/Installation-and-Setup.md`
4. Click "Save Page"

### 4. Upload Getting-Started

1. Click "New Page"
2. Title: `Getting-Started`
3. Paste content from `docs/wiki/Getting-Started.md`
4. Click "Save Page"

### 5. Upload Scenario-Formats

1. Click "New Page"
2. Title: `Scenario-Formats`
3. Paste content from `docs/wiki/Scenario-Formats.md`
4. Click "Save Page"

### 6. Upload Template-Variables

1. Click "New Page"
2. Title: `Template-Variables`
3. Paste content from `docs/wiki/Template-Variables.md`
4. Click "Save Page"

### 7. Upload API-Reference

1. Click "New Page"
2. Title: `API-Reference`
3. Paste content from `docs/wiki/API-Reference.md`
4. Click "Save Page"

## Quick Reference: File to Page Mapping

| File | Wiki Page Title |
|------|----------------|
| `Home.md` | `Home` |
| `_Sidebar.md` | `_Sidebar` |
| `Installation-and-Setup.md` | `Installation-and-Setup` |
| `Getting-Started.md` | `Getting-Started` |
| `Scenario-Formats.md` | `Scenario-Formats` |
| `Template-Variables.md` | `Template-Variables` |
| `API-Reference.md` | `API-Reference` |

**Note:** Don't upload `README.md` - that's just documentation for you about the wiki files.

## Verifying Your Wiki

After uploading, navigate to:
```
https://github.com/reaandrew/politest/wiki
```

You should see:
- Home page with overview and quick start
- Sidebar navigation on the right
- All pages accessible via sidebar links

## Need Help?

If you get stuck:
1. Make sure you clicked "Create the first page" in Step 1
2. The wiki repository won't exist until you create at least one page manually
3. Once the first page exists, you can use git to upload the rest

---

**Ready?** Start with [Step 1](#step-1-create-the-first-wiki-page) above!
