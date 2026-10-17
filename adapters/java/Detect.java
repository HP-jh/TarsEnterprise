// TarsEnterprise Java Adapter — Environment Detection & Tool Listing
import java.util.*;

public class Detect {
    public static void main(String[] args) {
        Map<String, Object> info = new LinkedHashMap<>();
        info.put("lang", "java");
        info.put("version", System.getProperty("java.version"));
        info.put("vendor", System.getProperty("java.vendor"));
        info.put("home", System.getProperty("java.home"));
        info.put("tools", Arrays.asList("javac", "jar", "javadoc"));
        System.out.println(toJson(info));
    }

    static String toJson(Map<String, Object> map) {
        StringBuilder sb = new StringBuilder("{");
        boolean first = true;
        for (Map.Entry<String, Object> e : map.entrySet()) {
            if (!first) sb.append(",");
            first = false;
            sb.append("\"").append(e.getKey()).append("\":");
            if (e.getValue() instanceof List) {
                sb.append("[");
                List<?> list = (List<?>) e.getValue();
                for (int i = 0; i < list.size(); i++) {
                    if (i > 0) sb.append(",");
                    sb.append("\"").append(list.get(i)).append("\"");
                }
                sb.append("]");
            } else {
                sb.append("\"").append(e.getValue()).append("\"");
            }
        }
        sb.append("}");
        return sb.toString();
    }
}
