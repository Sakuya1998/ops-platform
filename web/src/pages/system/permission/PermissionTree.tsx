import { useState, useEffect } from "react";
import { Tree, Input, Spin } from "antd";
import { api } from "../../../services/api";

export default function PermissionTree() {
  const [treeData, setTreeData] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchValue, setSearchValue] = useState("");

  const fetchPermissions = async () => {
    setLoading(true);
    try {
      const res = await api.get("/permissions");
      setTreeData(res.data.data || []);
    } catch (e) { console.error(e); }
    finally { setLoading(false); }
  };

  useEffect(() => { fetchPermissions(); }, []);

  return (
    <div>
      <h2 style={{ marginBottom: 16 }}>权限管理</h2>
      <Input.Search
        placeholder="搜索权限"
        style={{ width: 300, marginBottom: 16 }}
        onChange={(e) => setSearchValue(e.target.value)}
      />
      {loading ? (
        <Spin />
      ) : (
        <Tree
          treeData={treeData}
          fieldNames={{ title: "name", key: "id" }}
          defaultExpandAll
          showLine
        />
      )}
    </div>
  );
}
