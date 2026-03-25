-- ========== gallery_assets: Table Creation ==========

CREATE TABLE content.gallery_assets (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id UUID NOT NULL,
    workspace_id UUID NOT NULL,
    key VARCHAR(500) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    size BIGINT NOT NULL,
    sha256 CHAR(64) NOT NULL,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT fk_gallery_assets_tenant FOREIGN KEY (tenant_id) REFERENCES tenancy.tenants(id) ON DELETE CASCADE,
    CONSTRAINT fk_gallery_assets_workspace FOREIGN KEY (workspace_id) REFERENCES tenancy.workspaces(id) ON DELETE CASCADE,
    CONSTRAINT fk_gallery_assets_user FOREIGN KEY (created_by) REFERENCES identity.users(id)
);

-- ========== gallery_assets: Indexes ==========

CREATE UNIQUE INDEX idx_gallery_assets_key ON content.gallery_assets (workspace_id, key);
CREATE UNIQUE INDEX idx_gallery_assets_sha256 ON content.gallery_assets (workspace_id, sha256);
CREATE INDEX idx_gallery_assets_workspace ON content.gallery_assets (workspace_id);
CREATE INDEX idx_gallery_assets_filename ON content.gallery_assets (workspace_id, filename);
CREATE INDEX idx_gallery_assets_created_at ON content.gallery_assets (workspace_id, created_at DESC);
