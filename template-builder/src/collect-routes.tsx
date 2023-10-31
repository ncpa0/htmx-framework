import { renderToHtmlAsync } from "jsxte";
import { appCtx } from "./router";
import { pathCmp } from "./utils/paths";

type RouteDefinition = {
  path: string;
  containerID: string;
};

class RouteCollection {
  private routes: Array<RouteDefinition> = [];
  public topRouter: string = "";

  public add(route: RouteDefinition) {
    this.routes.push(route);
  }

  public get(path: string): RouteDefinition | undefined {
    return this.routes.find((r) => pathCmp(r.path, path));
  }

  public getAll(): Array<RouteDefinition> {
    return this.routes;
  }

  public has(path: string): boolean {
    return this.routes.some((r) => pathCmp(r.path, path));
  }

  public concatInto(collection: RouteCollection) {
    this.routes = this.routes.concat(collection.routes);
  }
}

export const collectRoutes = async (
  tree: JSX.Element,
  selectedRoute: string[] = [],
  collection: RouteCollection = new RouteCollection()
): Promise<RouteCollection> => {
  const newRoutes: Array<RouteDefinition> = [];

  const registerRoute = (path: string, routerContainerId: string) => {
    if (collection.has(path)) {
      return;
    }
    const route = {
      path,
      containerID: routerContainerId,
    };
    newRoutes.push(route);
    collection.add(route);
  };

  const getRouteContainerId = (path: string): string => {
    return collection.get(path)?.containerID ?? collection.topRouter;
  };

  const addRouter = (routerContainerId: string) => {
    if (!collection.topRouter) {
      collection.topRouter = routerContainerId;
    }
  };

  await renderToHtmlAsync(
    <appCtx.Provider
      value={{
        selectedRoute,
        currentRoute: [],
        registerRoute,
        getRouteContainerId,
        addRouter,
      }}
    >
      {tree}
    </appCtx.Provider>
  );

  for (const r of newRoutes) {
    await collectRoutes(tree, r.path.split("/").filter(Boolean), collection);
  }

  return collection;
};
